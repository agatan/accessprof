package accessprof

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var testHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Path: %s", r.URL.Path)
})

func TestAccessProf_Wrap_recordsRequests(t *testing.T) {
	var accessProf AccessProf
	server := httptest.NewServer(accessProf.Wrap(testHandler))
	defer server.Close()

	http.Get(server.URL + "/get/1")
	http.Get(server.URL + "/get/2")
	http.Post(server.URL+"/post/1", "application/json", strings.NewReader(`{"test": "post"}`))

	if accessProf.Count() != 3 {
		t.Fatalf("The server got 3 requests, but got %d access logs", accessProf.Count())
	}
}

func TestAccessProf_Report_aggregatesByMethod(t *testing.T) {
	var a AccessProf
	server := httptest.NewServer(a.Wrap(testHandler))
	defer server.Close()

	http.Get(server.URL)
	http.Post(server.URL, "", nil)

	report := a.Report()
	if len(report.Segments) != 2 {
		t.Fatalf("expected 2 report segments, GET / and POST /; but got %d", len(report.Segments))
	}
}
