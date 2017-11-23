package accessprof

import (
	"bytes"
	"math"
	"strconv"
	"time"

	"github.com/olekukonko/tablewriter"
)

type ReportSegment struct {
	Method     string
	Path       string
	Status     int
	AccessLogs []*AccessLog
}

func (seg *ReportSegment) match(l *AccessLog) bool {
	return seg.Method == l.Method && seg.Path == l.Path && seg.Status == l.Status
}

func (seg *ReportSegment) add(l *AccessLog) {
	seg.AccessLogs = append(seg.AccessLogs, l)
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
	w.SetHeader([]string{"Status", "Method", "Path", "Count", "MIN", "MAX", "SUM", "AVG", "MIN(BODY)", "MAX(BODY)", "Sum(body)", "AVG(BODY)"})
	for _, seg := range r.Segments {
		w.Append([]string{
			strconv.Itoa(seg.Status),
			seg.Method,
			seg.Path,
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
