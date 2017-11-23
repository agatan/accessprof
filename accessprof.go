package accessprof

import (
	"net/http"
	"regexp"
	"sync"
	"time"
)

var defaultAccessProf AccessProf

func Wrap(h http.Handler) http.Handler {
	return defaultAccessProf.Wrap(h)
}

func Count() int {
	return defaultAccessProf.Count()
}

func MakeReport(aggregates []*regexp.Regexp) *Report {
	return defaultAccessProf.MakeReport(aggregates)
}

func Reset() {
	defaultAccessProf.Reset()
}

type AccessLog struct {
	Method           string
	Path             string
	RequestBodySize  int64
	Status           int
	ResponseBodySize int
	ResponseTime     time.Duration
}

type AccessProf struct {
	mu         sync.Mutex
	accessLogs []*AccessLog
}

func (a *AccessProf) Wrap(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := &AccessLog{
			Method:          r.Method,
			Path:            r.URL.Path,
			RequestBodySize: r.ContentLength,
		}
		start := time.Now()
		wrapped := responseWriter{w: w}
		h.ServeHTTP(&wrapped, r)
		l.ResponseTime = time.Now().Sub(start)
		l.Status = wrapped.status
		l.ResponseBodySize = wrapped.writtenSize
		a.mu.Lock()
		a.accessLogs = append(a.accessLogs, l)
		a.mu.Unlock()
	})
}

func (a *AccessProf) Count() int {
	a.mu.Lock()
	n := len(a.accessLogs)
	a.mu.Unlock()
	return n
}

func (a *AccessProf) MakeReport(aggregates []*regexp.Regexp) *Report {
	var segs []*ReportSegment
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, l := range a.accessLogs {
		newsegment := true
		for _, seg := range segs {
			if seg.match(l) {
				seg.add(l)
				newsegment = false
				break
			}
		}
		if newsegment {
			seg := &ReportSegment{
				Method:     l.Method,
				Path:       l.Path,
				Status:     l.Status,
				AccessLogs: []*AccessLog{l},
			}
			for _, agg := range aggregates {
				if agg.MatchString(l.Path) {
					seg.PathRegexp = agg
					break
				}
			}
			segs = append(segs, seg)
		}
	}

	return &Report{Segments: segs}
}

func (a *AccessProf) Reset() {
	a.mu.Lock()
	a.accessLogs = a.accessLogs[:0]
	a.mu.Unlock()
}
