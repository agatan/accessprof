package accessprof

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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
	http.Post(server.URL, "application/json", strings.NewReader("{}"))
	http.Post(server.URL, "application/json", strings.NewReader(`{"key": "value"}`))

	report := a.Report()
	if len(report.Segments) != 2 {
		t.Fatalf("expected 2 report segments, GET / and POST /; but got %d", len(report.Segments))
	}
}

func TestAccessProf_Report_aggregatesByPath(t *testing.T) {
	var a AccessProf
	server := httptest.NewServer(a.Wrap(testHandler))
	defer server.Close()

	http.Get(server.URL)
	http.Get(server.URL + "/test")
	http.Get(server.URL + "/test")

	report := a.Report()
	if len(report.Segments) != 2 {
		t.Fatalf("expected 2 report segments, GET / and GET /test; but got %d", len(report.Segments))
	}
}

func ExampleAccessProf() {
	var a AccessProf
	server := httptest.NewServer(a.Wrap(testHandler))
	defer server.Close()

	http.Get(server.URL)
	http.Get(server.URL + "/test")
	http.Get(server.URL + "/test")
	http.Post(server.URL, "application/json", strings.NewReader("{}"))
	http.Post(server.URL, "application/json", strings.NewReader(`{"key": "value"}`))

	report := a.Report()
	fmt.Println(report.String())

	// Output:
	// +--------+--------+-------+-------+---------+----------+----------+----------+-----------+-----------+-----------+-----------+
	// | STATUS | METHOD | PATH  | COUNT |   MIN   |   MAX    |   SUM    |   AVG    | MIN(BODY) | MAX(BODY) | SUM(BODY) | AVG(BODY) |
	// +--------+--------+-------+-------+---------+----------+----------+----------+-----------+-----------+-----------+-----------+
	// |    200 | GET    | /     |     1 | 2.246µs | 2.246µs  | 2.246µs  | 2.246µs  |         8 |         8 |         8 |     8.000 |
	// |    200 | GET    | /test |     2 | 1.858µs | 2.077µs  | 3.935µs  | 1.967µs  |        12 |        12 |        24 |    12.000 |
	// |    200 | POST   | /     |     2 | 9.709µs | 10.463µs | 20.172µs | 10.086µs |        10 |        24 |        34 |    17.000 |
	// +--------+--------+-------+-------+---------+----------+----------+----------+-----------+-----------+-----------+-----------+

}
