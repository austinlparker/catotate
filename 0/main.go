package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/lightstep/lightstep-tracer-go"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
)

var (
	serverPort   = ":3001"
	catString    = "hello, cat!"
	traceVerbose = os.Getenv("TRACE_LEVEL") == "local"
)

func main() {
	tracer := lightstep.NewTracer(lightstep.Options{
		Collector: lightstep.Endpoint{
			Host:      "localhost",
			Port:      8360,
			Plaintext: true,
		},
		AccessToken: "dev",
	})
	opentracing.SetGlobalTracer(tracer)

	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("../static"))
	mux.Handle("/", fs)
	mw := nethttp.Middleware(tracer, mux)
	log.Printf("Server listening on port %s", serverPort)
	http.ListenAndServe(serverPort, mw)
}

func getCat(ctx context.Context) {
	ctx, localSpan := withLocalSpan(ctx)
	// do stuff
	finishLocalSpan(localSpan)
}

func annotateCat() {

}

func withLocalSpan(ctx context.Context) (context.Context, opentracing.Span) {
	if traceVerbose {
		pc, _, _, ok := runtime.Caller(1)
		callerFn := runtime.FuncForPC(pc)
		if ok && callerFn != nil {
			span, ctx := opentracing.StartSpanFromContext(ctx, callerFn.Name())
			return ctx, span
		}
	}
	return ctx, opentracing.SpanFromContext(ctx)
}

func finishLocalSpan(span opentracing.Span) {
	if traceVerbose {
		span.Finish()
	}
}
