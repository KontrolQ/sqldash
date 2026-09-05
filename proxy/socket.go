package proxy

import (
	"context"
	"net/http"
	"sync"
	"time"

	"sqldash/config"
	"sqldash/proxy/hrana"
	"sqldash/telemetry"
	"sqldash/utils/logger"

	"github.com/coder/websocket"
)

type pending struct {
	statements []string
	startedAt  time.Time
}

func carrySocket(writer http.ResponseWriter, request *http.Request, database string) {
	fromClient, acceptError := websocket.Accept(writer, request, &websocket.AcceptOptions{
		Subprotocols:   hrana.Subprotocols,
		OriginPatterns: config.Access.AllowedOrigins,
	})

	if acceptError != nil {
		logger.Warnf(LogPrefix, SocketRefusedLog, acceptError)
		return
	}

	defer fromClient.CloseNow()

	runContext, stop := context.WithCancel(request.Context())
	defer stop()

	toServer, _, dialError := websocket.Dial(runContext, SocketScheme+config.Sqld.Address+"/", &websocket.DialOptions{
		Subprotocols: []string{fromClient.Subprotocol()},
		HTTPHeader:   http.Header{NamespaceHeader: []string{database}},
	})

	if dialError != nil {
		logger.Errorf(LogPrefix, SocketDialFailedLog, dialError)
		fromClient.Close(websocket.StatusInternalError, DatabaseDown)
		return
	}

	defer toServer.CloseNow()

	waiting := &sync.Map{}
	finished := make(chan struct{}, 2)

	go carryFrames(runContext, fromClient, toServer, database, waiting, true, finished)
	go carryFrames(runContext, toServer, fromClient, database, waiting, false, finished)

	<-finished
}

func carryFrames(runContext context.Context, from *websocket.Conn, to *websocket.Conn, database string, waiting *sync.Map, fromClient bool, finished chan<- struct{}) {
	defer func() { finished <- struct{}{} }()

	from.SetReadLimit(MaximumFrame)

	for {
		kind, frame, readError := from.Read(runContext)
		if readError != nil {
			return
		}

		if kind == websocket.MessageText {
			if fromClient {
				noteAsked(waiting, frame)
			} else {
				noteHeld(waiting, database, frame)
			}
		}

		if writeError := to.Write(runContext, kind, frame); writeError != nil {
			return
		}
	}
}

func noteAsked(waiting *sync.Map, frame []byte) {
	identifier, statements := hrana.StatementsSent(frame)

	if len(statements) == 0 {
		return
	}

	waiting.Store(identifier, &pending{statements: statements, startedAt: time.Now()})
}

func noteHeld(waiting *sync.Map, database string, frame []byte) {
	identifier, measured, matched := hrana.StatementsHeld(frame)
	if !matched {
		return
	}

	held, seen := waiting.LoadAndDelete(identifier)
	if !seen {
		return
	}

	asked := held.(*pending)

	for index, statement := range measured {
		sql := ""
		if index < len(asked.statements) {
			sql = asked.statements[index]
		} else if len(asked.statements) > 0 {
			sql = asked.statements[len(asked.statements)-1]
		}

		telemetry.Record(telemetry.Observation{
			DatabaseName: database,
			SQL:          sql,
			DurationMs:   statement.DurationMs,
			RowsRead:     statement.RowsRead,
			RowsWritten:  statement.RowsWritten,
			RowsReturned: statement.RowsReturned,
			Failed:       statement.Failed,
			OccurredAt:   asked.startedAt,
		})
	}
}
