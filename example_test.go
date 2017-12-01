package accessprof_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"time"

	"github.com/agatan/accessprof"
	"github.com/agatan/timejump"
)

var exampleHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(fmt.Sprintf("Path: %s\n", r.URL.Path)))
	io.Copy(w, r.Body)
	r.Body.Close()
})

func Example() {
	// Use timejump package to mock `time.Now`.
	timejump.Activate()
	defer timejump.Deactivate()
	timejump.Stop()
	timejump.Jump(time.Date(2017, 12, 2, 0, 0, 0, 0, time.UTC))

	var a accessprof.AccessProf
	handler := a.Wrap(exampleHandler, "")
	server := httptest.NewServer(handler)
	defer server.Close()

	http.Get(server.URL)
	http.Get(server.URL + "/test/123")
	http.Get(server.URL + "/test/456")
	http.Post(server.URL+"/test/123", "application/json", strings.NewReader("{}"))
	http.Post(server.URL+"/test/789", "application/json", strings.NewReader(`{"key": "value"}`))

	report := handler.Report([]*regexp.Regexp{
		regexp.MustCompile(`/test/\d+`),
	})
	fmt.Print(report.String())

	// Output:
	// +--------+--------+-----------+-------+-----+-----+-----+-----+-----------+-----------+-----------+-----------+
	// | STATUS | METHOD |   PATH    | COUNT | MIN | MAX | SUM | AVG | MIN(BODY) | MAX(BODY) | SUM(BODY) | AVG(BODY) |
	// +--------+--------+-----------+-------+-----+-----+-----+-----+-----------+-----------+-----------+-----------+
	// |    200 | GET    | /         |     1 | 0s  | 0s  | 0s  | 0s  |         8 |         8 |         8 |     8.000 |
	// |    200 | GET    | /test/\d+ |     2 | 0s  | 0s  | 0s  | 0s  |        16 |        16 |        32 |    16.000 |
	// |    200 | POST   | /test/\d+ |     2 | 0s  | 0s  | 0s  | 0s  |        18 |        32 |        50 |    25.000 |
	// +--------+--------+-----------+-------+-----+-----+-----+-----+-----------+-----------+-----------+-----------+
}
