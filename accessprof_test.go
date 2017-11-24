package accessprof

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

var testHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(fmt.Sprintf("Path: %s\n", r.URL.Path)))
	io.Copy(w, r.Body)
	r.Body.Close()
})

func TestAccessProf_Wrap_recordsRequests(t *testing.T) {
	var accessProf AccessProf
	server := httptest.NewServer(accessProf.Wrap(testHandler, ""))
	defer server.Close()

	http.Get(server.URL + "/get/1")
	http.Get(server.URL + "/get/2")
	http.Post(server.URL+"/post/1", "application/json", strings.NewReader(`{"test": "post"}`))

	if accessProf.Count() != 3 {
		t.Fatalf("The server got 3 requests, but got %d access logs", accessProf.Count())
	}
}

func TestAccessProf_MakeReport_aggregatesByMethod(t *testing.T) {
	var a AccessProf
	server := httptest.NewServer(a.Wrap(testHandler, ""))
	defer server.Close()

	http.Get(server.URL)
	http.Post(server.URL, "application/json", strings.NewReader("{}"))
	http.Post(server.URL, "application/json", strings.NewReader(`{"key": "value"}`))

	report := a.Report(nil)
	if len(report.Segments) != 2 {
		t.Fatalf("expected 2 report segments, GET / and POST /; but got %d", len(report.Segments))
	}
}

func TestAccessProf_MakeReport_aggregatesByPath(t *testing.T) {
	var a AccessProf
	server := httptest.NewServer(a.Wrap(testHandler, ""))
	defer server.Close()

	http.Get(server.URL)
	http.Get(server.URL + "/test")
	http.Get(server.URL + "/test")

	report := a.Report(nil)
	if len(report.Segments) != 2 {
		t.Fatalf("expected 2 report segments, GET / and GET /test; but got %d", len(report.Segments))
	}
}

func TestAccessProf_MakeReport_aggregatesByPathRegexp(t *testing.T) {
	var a AccessProf
	server := httptest.NewServer(a.Wrap(testHandler, ""))
	defer server.Close()

	http.Get(server.URL)
	http.Get(server.URL + "/test/123")
	http.Get(server.URL + "/test/456")
	http.Post(server.URL+"/test/789", "application/json", strings.NewReader(`{"key": "value"}`))

	report := a.Report([]*regexp.Regexp{regexp.MustCompile(`/test/\d+`)})
	if len(report.Segments) != 3 {
		t.Fatalf("expected 3 report segments, GET /, GET /test/\\d+ and POST /test/\\d+; but got %d", len(report.Segments))
	}
}

func TestAccessProf_Reset(t *testing.T) {
	var a AccessProf
	server := httptest.NewServer(a.Wrap(testHandler, ""))
	defer server.Close()

	http.Get(server.URL)
	http.Get(server.URL + "/test")
	http.Get(server.URL + "/test")

	report := a.Report(nil)
	if len(report.Segments) != 2 {
		t.Fatalf("expected 2 report segments, GET / and GET /test; but got %d", len(report.Segments))
	}

	a.Reset()
	report = a.Report(nil)
	if len(report.Segments) != 0 {
		t.Fatalf("Reset does not work")
	}
}

func TestAccessProf_ServeHTTP_withoutReportPath(t *testing.T) {
	reached := false
	var a AccessProf
	server := httptest.NewServer(a.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
	}), ""))
	defer server.Close()

	http.Get(server.URL)

	if !reached {
		t.Fatal("All requests should be reached to the original handler if ReportPath is not specified")
	}
}

func TestAccessProf_ServeHTTP_resetLogs(t *testing.T) {
	var a AccessProf
	server := httptest.NewServer(a.Wrap(testHandler, "/accessprof"))
	defer server.Close()

	http.Get(server.URL)
	http.Get(server.URL + "/test")

	if a.Count() != 2 {
		t.Fatal("non-report requests should be reached to the original handler")
	}

	req, _ := http.NewRequest(http.MethodDelete, server.URL+"/accessprof", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Errorf("Reset request is failed: error %v, status code %d", err, resp.StatusCode)
	}

	if a.Count() != 0 {
		t.Fatalf("Reset request is failed")
	}
}

func TestAccessProf_Report_DumpToFile(t *testing.T) {
	a := AccessProf{LogFile: "ltsv"}
	defer os.Remove("ltsv")
	server := httptest.NewServer(a.Wrap(testHandler, ""))
	defer server.Close()

	for i := 0; i < 200; i++ {
		http.Get(server.URL)
	}
	for i := 0; i < 100; i++ {
		http.Get(server.URL + "/test")
	}

	report := a.Report(nil)
	if len(report.Segments) != 2 {
		t.Fatalf("Report should read from dump file, but it doesn't work: got %d segments", len(report.Segments))
	}
}
