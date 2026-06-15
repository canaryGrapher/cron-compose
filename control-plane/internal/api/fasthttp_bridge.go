package api

import (
	"net/http"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

// fasthttpHTTPHandler adapts a stdlib net/http.Handler (e.g. promhttp) so it can be
// called from Fiber v3 handlers, which use fasthttp under the hood.
func fasthttpHTTPHandler(h http.Handler) fasthttp.RequestHandler {
	return fasthttpadaptor.NewFastHTTPHandler(h)
}
