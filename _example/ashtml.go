package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/agatan/accessprof"
)

var exampleHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(fmt.Sprintf("Path: %s\n", r.URL.Path)))
	io.Copy(w, r.Body)
	r.Body.Close()
})

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/accessprof/", func(w http.ResponseWriter, _ *http.Request) {
		if err := accessprof.MakeReport(nil).RenderHTML(w); err != nil {
			panic(err)
		}
	})
	mux.Handle("/", exampleHandler)

	if err := http.ListenAndServe(":8080", accessprof.Wrap(mux)); err != nil {
		panic(err)
	}
}
