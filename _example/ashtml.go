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
	var a accessprof.AccessProf
	handler := a.Wrap(exampleHandler, "/accessprof")
	// If you want to save memory, use LogFile to dump logs to the file.
	// handler := &accessprof.Handler{Handler: exampleHandler, ReportPath: "/accessprof", LogFile: "accessprof.ltsv"}

	if err := http.ListenAndServe(":8080", handler); err != nil {
		panic(err)
	}
}
