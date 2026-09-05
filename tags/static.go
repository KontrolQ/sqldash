package tags

import (
	"fmt"

	"github.com/flosch/pongo2/v6"
)

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
	_, writeError := writer.WriteString(fmt.Sprintf("%s/%s", StaticPrefix, self.Path))
	if writeError != nil {
		return &pongo2.Error{
			Sender:    "tag:static",
			OrigError: fmt.Errorf(TemplateWriteFailed),
		}
	}

	return nil
}
