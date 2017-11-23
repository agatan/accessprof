package accessprof

import (
	"net/http"
	"sync"
	"time"
)

var defaultAccessProf AccessProf

func Wrap(h http.Handler) http.Handler {
	return defaultAccessProf.Wrap(h)
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
		h.ServeHTTP(w, r)
		l.ResponseTime = time.Now().Sub(start)
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

func (a *AccessProf) Report() *Report {
	var segs []*ReportSegment
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, l := range a.accessLogs {
		for _, seg := range segs {
			if seg.match(l) {
				seg.add(l)
				goto next
			}
		}
		segs = append(segs, &ReportSegment{
			Method:     l.Method,
			Path:       l.Path,
			Status:     l.Status,
			AccessLogs: []*AccessLog{l},
		})
	next:
	}

	return &Report{Segments: segs}
}

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

type Report struct {
	Segments []*ReportSegment
}
