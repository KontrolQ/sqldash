package tags

import (
	"fmt"
	"strings"

	"sqldash/utils/collections"
	"sqldash/utils/urls"

	"github.com/flosch/pongo2/v6"
)

type UrlNode struct {
	RouteName    string
	Params       collections.Record[string, pongo2.IEvaluator]
	VariableName string
}

func url(document *pongo2.Parser, start *pongo2.Token, arguments *pongo2.Parser) (pongo2.INodeTag, *pongo2.Error) {
	routeNameToken := arguments.MatchType(pongo2.TokenString)
	if routeNameToken == nil {
		return nil, arguments.Error(ExpectedRouteName, nil)
	}

	params := make(collections.Record[string, pongo2.IEvaluator])

	var variableName string

	for arguments.Remaining() > 0 {
		if arguments.Match(pongo2.TokenKeyword, "as") != nil {
			nameToken := arguments.MatchType(pongo2.TokenIdentifier)
			if nameToken == nil {
				return nil, arguments.Error(ExpectedVariableName, nil)
			}
			variableName = nameToken.Val
			break
		}

		keyToken := arguments.MatchType(pongo2.TokenIdentifier)
		if keyToken == nil {
			return nil, arguments.Error(ExpectedParamKey, nil)
		}

		if arguments.Match(pongo2.TokenSymbol, "=") == nil {
			return nil, arguments.Error(ExpectedEquals, nil)
		}

		valueExpression, parseError := arguments.ParseExpression()
		if parseError != nil {
			return nil, parseError
		}

		params[keyToken.Val] = valueExpression
	}

	return &UrlNode{
		RouteName:    routeNameToken.Val,
		Params:       params,
		VariableName: variableName,
	}, nil
}

func (self *UrlNode) Execute(executionContext *pongo2.ExecutionContext, writer pongo2.TemplateWriter) *pongo2.Error {
	path, exists := urls.GetFullPath(self.RouteName)
	if !exists {
		return &pongo2.Error{
			Sender:    "tag:url",
			OrigError: fmt.Errorf(RouteNotFound, self.RouteName),
		}
	}

	for key, expression := range self.Params {
		evaluatedValue, evaluationError := expression.Evaluate(executionContext)
		if evaluationError != nil {
			return evaluationError
		}

		placeholder := fmt.Sprintf(":%s", key)
		replacement := fmt.Sprintf("%v", evaluatedValue.Interface())
		path = strings.ReplaceAll(path, placeholder, replacement)
	}

	if self.VariableName != "" {
		executionContext.Public[self.VariableName] = path
		return nil
	}

	_, writeError := writer.WriteString(path)
	if writeError != nil {
		return &pongo2.Error{
			Sender:    "tag:url",
			OrigError: fmt.Errorf(TemplateWriteFailed),
		}
	}

	return nil
}
