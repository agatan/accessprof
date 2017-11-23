# accessprof: Wrap http.Handler to analyze accesses

[![GoDoc](https://godoc.org/github.com/agatan/accessprof?status.svg)](https://godoc.org/github.com/agatan/accessprof)

```sh
go get github.com/agatan/accessprof
```

## Example

See `_example/example.go`.

```go
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
}
```

```
+--------+--------+-----------+-------+----------+----------+----------+----------+-----------+-----------+-----------+-----------+
| STATUS | METHOD |   PATH    | COUNT |   MIN    |   MAX    |   SUM    |   AVG    | MIN(BODY) | MAX(BODY) | SUM(BODY) | AVG(BODY) |
+--------+--------+-----------+-------+----------+----------+----------+----------+-----------+-----------+-----------+-----------+
|    200 | GET    | /         |     1 | 15.195µs | 15.195µs | 15.195µs | 15.195µs |         8 |         8 |         8 |     8.000 |
|    200 | GET    | /test/\d+ |     2 | 4.61µs   | 10.659µs | 15.269µs | 7.634µs  |        16 |        16 |        32 |    16.000 |
|    200 | POST   | /test/\d+ |     2 | 10.885µs | 22.104µs | 32.989µs | 16.494µs |        18 |        32 |        50 |    25.000 |
+--------+--------+-----------+-------+----------+----------+----------+----------+-----------+-----------+-----------+-----------+
```
