package meta

import (
	"sqldash/utils/logger"

	"github.com/gofiber/fiber/v3"
)

type Param struct {
	Key   string
	Value string
}

type RequestInfo struct {
	Path        string
	Method      string
	Query       []Param
	Headers     []Param
	QueryString string
	IP          string
	URL         string
}

type RequestData struct {
	RequestInfo
	Context fiber.Ctx
}

func BuildRequest(context fiber.Ctx) RequestInfo {
	return RequestInfo{
		Path:        context.Path(),
		Method:      context.Method(),
		Query:       buildQueryParams(context),
		Headers:     buildHeaders(context),
		QueryString: string(context.RequestCtx().Request.URI().QueryString()),
		IP:          context.IP(),
		URL:         context.OriginalURL(),
	}
}

func Request(context fiber.Ctx) *RequestData {
	data, held := context.Locals(RequestKey).(RequestInfo)
	if !held {
		logger.Errorf(LogPrefix, RequestContextMissing)
		return nil
	}

	return &RequestData{
		RequestInfo: data,
		Context:     context,
	}
}

func (self *RequestData) Param(key string) string {
	if self == nil || self.Context == nil {
		return ""
	}

	return self.Context.Params(key)
}

func (self *RequestData) Query(key string) string {
	if self == nil {
		return ""
	}

	return findParam(self.RequestInfo.Query, key)
}

func (self *RequestData) Header(key string) string {
	if self == nil {
		return ""
	}

	return findParam(self.RequestInfo.Headers, key)
}

func buildQueryParams(context fiber.Ctx) []Param {
	params := make([]Param, 0)

	context.RequestCtx().Request.URI().QueryArgs().VisitAll(func(name []byte, value []byte) {
		params = append(params, Param{Key: string(name), Value: string(value)})
	})

	return params
}

func buildHeaders(context fiber.Ctx) []Param {
	params := make([]Param, 0)

	context.RequestCtx().Request.Header.VisitAll(func(name []byte, value []byte) {
		params = append(params, Param{Key: string(name), Value: string(value)})
	})

	return params
}

func findParam(params []Param, key string) string {
	for _, param := range params {
		if param.Key == key {
			return param.Value
		}
	}

	return ""
}
