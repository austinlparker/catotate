package main

import (
	"context"
	"log"
	"net/http"

	"github.com/lightstep/lightstep-tracer-go"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
)

var (
	serverPort = ":3001"
	catString  = "hello, cat!"
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

func getCat() {

}

func annotateCat() {

}

func withLocalSpan(ctx context.Context) (context.Context, opentracing.Span) {
	return nil, nil
}

func finishLocalSpan(span opentracing.Span) {

}
