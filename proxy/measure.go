package proxy

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"sqldash/proxy/hrana"
	"sqldash/telemetry"
)

type capturing struct {
	http.ResponseWriter
	body   bytes.Buffer
	status int
}

func (self *capturing) WriteHeader(status int) {
	self.status = status
	self.ResponseWriter.WriteHeader(status)
}

func (self *capturing) Write(payload []byte) (int, error) {
	if self.body.Len() < MaximumMeasuredBody {
		self.body.Write(payload)
	}

	return self.ResponseWriter.Write(payload)
}

func measure(writer http.ResponseWriter, request *http.Request, database string) {
	asked, readError := io.ReadAll(io.LimitReader(request.Body, MaximumMeasuredBody))
	request.Body.Close()

	if readError != nil {
		http.Error(writer, DatabaseDown, http.StatusBadRequest)
		return
	}

	request.Body = io.NopCloser(bytes.NewReader(asked))
	request.ContentLength = int64(len(asked))

	captured := &capturing{ResponseWriter: writer, status: http.StatusOK}
	startedAt := time.Now()

	toDatabase.ServeHTTP(captured, request)

	for _, statement := range hrana.Measure(asked, captured.body.Bytes()) {
		telemetry.Record(telemetry.Observation{
			DatabaseName: database,
			SQL:          statement.SQL,
			DurationMs:   statement.DurationMs,
			RowsRead:     statement.RowsRead,
			RowsWritten:  statement.RowsWritten,
			RowsReturned: statement.RowsReturned,
			Failed:       statement.Failed || captured.status >= http.StatusBadRequest,
			OccurredAt:   startedAt,
		})
	}
}
