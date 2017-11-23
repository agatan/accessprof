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
	handler := &accessprof.Handler{Handler: exampleHandler}
	mux := http.NewServeMux()
	mux.HandleFunc("/accessprof/", func(w http.ResponseWriter, _ *http.Request) {
		if err := handler.MakeReport(nil).RenderHTML(w); err != nil {
			panic(err)
		}
	})
	mux.Handle("/", handler)

	if err := http.ListenAndServe(":8080", mux); err != nil {
		panic(err)
	}
}
