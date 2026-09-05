package tags

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"sqldash/config"

	"github.com/flosch/pongo2/v6"
)

var stamps sync.Map

type StaticNode struct {
	Path string
}

func static(document *pongo2.Parser, start *pongo2.Token, arguments *pongo2.Parser) (pongo2.INodeTag, *pongo2.Error) {
	pathToken := arguments.MatchType(pongo2.TokenString)
	if pathToken == nil {
		return nil, arguments.Error(ExpectedStaticPath, nil)
	}

	return &StaticNode{Path: pathToken.Val}, nil
}

func (self *StaticNode) Execute(executionContext *pongo2.ExecutionContext, writer pongo2.TemplateWriter) *pongo2.Error {
	_, writeError := writer.WriteString(
		fmt.Sprintf(StaticAddressFormat, StaticPrefix, self.Path, stampFor(self.Path)),
	)

	if writeError != nil {
		return &pongo2.Error{
			Sender:    "tag:static",
			OrigError: fmt.Errorf(TemplateWriteFailed),
		}
	}

	return nil
}

func stampFor(path string) string {
	if !config.Server.Debug {
		if held, seen := stamps.Load(path); seen {
			return held.(string)
		}
	}

	stamp := config.AppVersion

	if held, statError := os.Stat(filepath.Join(StaticRoot, filepath.FromSlash(path))); statError == nil {
		stamp = strconv.FormatInt(held.ModTime().Unix(), 36)
	}

	stamps.Store(path, stamp)

	return stamp
}
