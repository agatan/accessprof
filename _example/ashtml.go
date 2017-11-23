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
	handler := &accessprof.Handler{Handler: exampleHandler, ReportPath: "/accessprof"}

	if err := http.ListenAndServe(":8080", handler); err != nil {
		panic(err)
	}
}
