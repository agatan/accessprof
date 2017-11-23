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
	server := httptest.NewServer(accessprof.Wrap(exampleHandler))
	defer server.Close()

	http.Get(server.URL)
	http.Get(server.URL + "/test/123")
	http.Get(server.URL + "/test/456")
	http.Post(server.URL+"/test/123", "application/json", strings.NewReader("{}"))
	http.Post(server.URL+"/test/789", "application/json", strings.NewReader(`{"key": "value"}`))

	report := accessprof.MakeReport([]*regexp.Regexp{
		regexp.MustCompile(`/test/\d+`),
	})
	fmt.Print(report.String())

	// Output:
	// +--------+--------+-------+-------+---------+----------+----------+----------+-----------+-----------+-----------+-----------+
	// | STATUS | METHOD | PATH  | COUNT |   MIN   |   MAX    |   SUM    |   AVG    | MIN(BODY) | MAX(BODY) | SUM(BODY) | AVG(BODY) |
	// +--------+--------+-------+-------+---------+----------+----------+----------+-----------+-----------+-----------+-----------+
	// |    200 | GET    | /     |     1 | 2.246µs | 2.246µs  | 2.246µs  | 2.246µs  |         8 |         8 |         8 |     8.000 |
	// |    200 | GET    | /test |     2 | 1.858µs | 2.077µs  | 3.935µs  | 1.967µs  |        12 |        12 |        24 |    12.000 |
	// |    200 | POST   | /     |     2 | 9.709µs | 10.463µs | 20.172µs | 10.086µs |        10 |        24 |        34 |    17.000 |
	// +--------+--------+-------+-------+---------+----------+----------+----------+-----------+-----------+-----------+-----------+
}
