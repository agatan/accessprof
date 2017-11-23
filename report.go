package accessprof

import (
	"bytes"
	"io"
	"math"
	"regexp"
	"strconv"
	"time"

	"github.com/alecthomas/template"
	"github.com/olekukonko/tablewriter"
)

type ReportSegment struct {
	Method     string
	Path       string
	PathRegexp *regexp.Regexp
	Status     int
	AccessLogs []*AccessLog
}

func (seg *ReportSegment) match(l *AccessLog) bool {
	if seg.Method != l.Method || seg.Status != l.Status {
		return false
	}
	if seg.PathRegexp != nil {
		return seg.PathRegexp.MatchString(l.Path)
	}
	return seg.Path == l.Path
}

func (seg *ReportSegment) add(l *AccessLog) {
	seg.AccessLogs = append(seg.AccessLogs, l)
}

func (seg *ReportSegment) AggregationPath() string {
	if seg.PathRegexp != nil {
		return seg.PathRegexp.String()
	}
	return seg.Path
}

func (seg *ReportSegment) Count() int {
	return len(seg.AccessLogs)
}

func (seg *ReportSegment) MinResponseTime() time.Duration {
	var n time.Duration = math.MaxInt32
	for _, l := range seg.AccessLogs {
		if n > l.ResponseTime {
			n = l.ResponseTime
		}
	}
	return n
}

func (seg *ReportSegment) MaxResponseTime() time.Duration {
	var n time.Duration
	for _, l := range seg.AccessLogs {
		if n < l.ResponseTime {
			n = l.ResponseTime
		}
	}
	return n
}

func (seg *ReportSegment) SumResponseTime() time.Duration {
	var n time.Duration
	for _, l := range seg.AccessLogs {
		n += l.ResponseTime
	}
	return n
}

func (seg *ReportSegment) AvgResponseTime() time.Duration {
	return seg.SumResponseTime() / time.Duration(seg.Count())
}

func (seg *ReportSegment) MinBody() int {
	var n int = math.MaxInt32
	for _, l := range seg.AccessLogs {
		if n > l.ResponseBodySize {
			n = l.ResponseBodySize
		}
	}
	return n
}

func (seg *ReportSegment) MaxBody() int {
	var n int
	for _, l := range seg.AccessLogs {
		if n < l.ResponseBodySize {
			n = l.ResponseBodySize
		}
	}
	return n
}

func (seg *ReportSegment) SumBody() int {
	var n int
	for _, l := range seg.AccessLogs {
		n += l.ResponseBodySize
	}
	return n
}

func (seg *ReportSegment) AvgBody() float64 {
	return float64(seg.SumBody()) / float64(seg.Count())
}

type Report struct {
	Segments []*ReportSegment
}

func (r *Report) String() string {
	var buf bytes.Buffer
	w := tablewriter.NewWriter(&buf)
	w.SetHeader([]string{"STATUS", "METHOD", "PATH", "COUNT", "MIN", "MAX", "SUM", "AVG", "MIN(BODY)", "MAX(BODY)", "SUM(BODY)", "AVG(BODY)"})
	for _, seg := range r.Segments {
		w.Append([]string{
			strconv.Itoa(seg.Status),
			seg.Method,
			seg.AggregationPath(),
			strconv.Itoa(seg.Count()),
			seg.MinResponseTime().String(),
			seg.MaxResponseTime().String(),
			seg.SumResponseTime().String(),
			seg.AvgResponseTime().String(),
			strconv.Itoa(seg.MinBody()),
			strconv.Itoa(seg.MaxBody()),
			strconv.Itoa(seg.SumBody()),
			strconv.FormatFloat(seg.AvgBody(), 'f', 3, 64),
		})
	}
	w.Render()
	return buf.String()
}

func (r *Report) RenderHTML(w io.Writer, reportPath string) error {
	data := struct {
		Header     []string
		Rows       [][]string
		ReportPath string
	}{}
	data.ReportPath = reportPath
	data.Header = []string{"STATUS", "METHOD", "PATH", "COUNT", "MIN", "MAX", "SUM", "AVG", "MIN(BODY)", "MAX(BODY)", "SUM(BODY)", "AVG(BODY)"}
	for _, seg := range r.Segments {
		data.Rows = append(data.Rows, []string{
			strconv.Itoa(seg.Status),
			seg.Method,
			seg.AggregationPath(),
			strconv.Itoa(seg.Count()),
			seg.MinResponseTime().String(),
			seg.MaxResponseTime().String(),
			seg.SumResponseTime().String(),
			seg.AvgResponseTime().String(),
			strconv.Itoa(seg.MinBody()),
			strconv.Itoa(seg.MaxBody()),
			strconv.Itoa(seg.SumBody()),
			strconv.FormatFloat(seg.AvgBody(), 'f', 3, 64),
		})
	}

	return htmlTemplate.Execute(w, data)
}

var htmlTemplate = template.Must(template.New("accessprof").Parse(`<!DOCTYPE html>
<html lang="ja">
  <head>
    <meta charset="UTF-8">
    <link rel="stylesheet" href="https://cdn.datatables.net/t/bs-3.3.6/jqc-1.12.0,dt-1.10.11/datatables.min.css"/>
    <script src="https://cdn.datatables.net/t/bs-3.3.6/jqc-1.12.0,dt-1.10.11/datatables.min.js"></script>
    <script>
      jQuery(function($){
        $("#foo-table").DataTable();
      });
    </script>
    <title></title>
  </head>
  <body>
    <table id="foo-table" class="table table-bordered">
      <thead>
        <tr>
          {{ range .Header }}
            <th>{{.}}</th>
          {{ end }}
        </tr>
      </thead>
      <tbody>
        {{ range .Rows }}
          <tr>
            {{ range . }}
              <td>{{ . }}</td>
            {{end}}
          </tr>
        {{ end }}
      </tbody>
    </table>
    <form action="{{ .ReportPath }}" method="get">
	  <input type="text" name="agg" placeholder="/users/\d+,/.*\.png">
	  <input type="submit" value="Go">
    </form>
  </body>
</html>
`))
