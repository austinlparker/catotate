package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/lightstep/lightstep-tracer-go"
	"github.com/opentracing-contrib/go-stdlib/nethttp"
	"github.com/opentracing/opentracing-go"
)

const catString = `
 /\_/\
( o.o )
 > ^ <
`

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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, catString)
	})

	log.Fatal(http.ListenAndServe(":3000", nethttp.Middleware(tracer, http.DefaultServeMux)))
}

func getCat() {

}

func annotateCat() {

}
