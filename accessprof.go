package accessprof

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type AccessLog struct {
	Method           string
	Path             string
	RequestBodySize  int64
	Status           int
	ResponseBodySize int
	ResponseTime     time.Duration
}

type Handler struct {
	mu         sync.Mutex
	accessLogs []*AccessLog
	Handler    http.Handler
	ReportPath string
}

func (a *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if a.ReportPath != "" && r.URL.Path == a.ReportPath {
		if r.Method == http.MethodDelete {
			a.Reset()
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method == http.MethodGet {
			a.serveReportRequest(w, r)
			return
		}
	}
	l := &AccessLog{
		Method:          r.Method,
		Path:            r.URL.Path,
		RequestBodySize: r.ContentLength,
	}
	start := time.Now()
	wrapped := responseWriter{w: w}
	a.Handler.ServeHTTP(&wrapped, r)
	l.ResponseTime = time.Now().Sub(start)
	l.Status = wrapped.status
	l.ResponseBodySize = wrapped.writtenSize
	a.mu.Lock()
	a.accessLogs = append(a.accessLogs, l)
	a.mu.Unlock()
}

func (a *Handler) Count() int {
	a.mu.Lock()
	n := len(a.accessLogs)
	a.mu.Unlock()
	return n
}

func (a *Handler) Report(aggregates []*regexp.Regexp) *Report {
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

func (a *Handler) Reset() {
	a.mu.Lock()
	a.accessLogs = a.accessLogs[:0]
	a.mu.Unlock()
}

func (a *Handler) serveReportRequest(w http.ResponseWriter, r *http.Request) {
	if err := r.Body.Close(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	aggparam := r.URL.Query().Get("agg")
	var aggs []*regexp.Regexp
	if aggparam != "" {
		for _, agg := range strings.Split(aggparam, ",") {
			re, err := regexp.Compile(agg)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				body, _ := json.Marshal(map[string]string{
					"error": fmt.Sprintf("failed to compile regexp %q: %v", re, err),
				})
				w.Write(body)
				return
			}
			aggs = append(aggs, re)
		}
	}
	if err := a.Report(aggs).RenderHTML(w, a.ReportPath); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}
