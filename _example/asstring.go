package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"

	"github.com/agatan/accessprof"
)

var exampleHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(fmt.Sprintf("Path: %s\n", r.URL.Path)))
	io.Copy(w, r.Body)
	r.Body.Close()
})

func main() {
	handler := &accessprof.AccessProf{Handler: exampleHandler}
	server := httptest.NewServer(handler)
	defer server.Close()

	http.Get(server.URL)
	http.Get(server.URL + "/test/123")
	http.Get(server.URL + "/test/456")
	http.Post(server.URL+"/test/123", "application/json", strings.NewReader("{}"))
	http.Post(server.URL+"/test/789", "application/json", strings.NewReader(`{"key": "value"}`))

	report := handler.MakeReport([]*regexp.Regexp{
		regexp.MustCompile(`/test/\d+`),
	})
	fmt.Print(report.String())
}
