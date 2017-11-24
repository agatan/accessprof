package accessprof

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type AccessLog struct {
	Method           string
	Path             string
	RequestBodySize  int64
	Status           int
	ResponseBodySize int
	ResponseTime     time.Duration
	AccessedAt       time.Time
}

const (
	methodLabel           = "method"
	pathLabel             = "path"
	statusLabel           = "status"
	responseBodySizeLabel = "response_body_size"
	responseTimeLabel     = "response_time_nano"
	accessedAtLabel       = "accessed_at"
)

func (l *AccessLog) writeLTSV(w io.Writer) error {
	_, err := fmt.Fprintf(w, "%s:%s\t%s:%s\t%s:%d\t%s:%d\t%s:%d\t%s:%s\n",
		methodLabel, l.Method,
		pathLabel, l.Path,
		statusLabel, l.Status,
		responseBodySizeLabel, l.ResponseBodySize,
		responseTimeLabel, l.ResponseTime.Nanoseconds(),
		accessedAtLabel, l.AccessedAt.String(),
	)
	return errors.Wrap(err, "failed to write accesslog as ltsv")
}

type Handler struct {
	mu             sync.Mutex
	accessLogs     []*AccessLog
	Handler        http.Handler
	ReportPath     string
	LogFile        string
	FlushThreshold int
}

const (
	DefaultFlushThreshold = 1000
)

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
		AccessedAt:      time.Now(),
	}
	start := time.Now()
	wrapped := responseWriter{w: w}
	a.Handler.ServeHTTP(&wrapped, r)
	l.ResponseTime = time.Now().Sub(start)
	l.Status = wrapped.status
	l.ResponseBodySize = wrapped.writtenSize
	a.mu.Lock()
	a.accessLogs = append(a.accessLogs, l)
	if len(a.accessLogs) > a.FlushThreshold || a.FlushThreshold == 0 && len(a.accessLogs) > DefaultFlushThreshold {
		go a.flushLogs()
	}
	a.mu.Unlock()
}

func (a *Handler) Count() int {
	a.mu.Lock()
	n := len(a.accessLogs)
	a.mu.Unlock()
	return n
}

func (a *Handler) Report(aggregates []*regexp.Regexp) *Report {
	a.flushLogs()
	a.mu.Lock()
	defer a.mu.Unlock()

	var (
		segs  []*ReportSegment
		since time.Time
	)

	for _, l := range a.accessLogs {
		if since.IsZero() || since.After(l.AccessedAt) {
			since = l.AccessedAt
		}
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

	return &Report{Segments: segs, Aggregates: aggregates, Since: since}
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

func (a *Handler) flushLogs() (err error) {
	if a.LogFile == "" {
		return nil
	}

	a.mu.Lock()
	logs := a.accessLogs
	a.accessLogs = nil
	a.mu.Unlock()

	f, err := os.OpenFile(a.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to open log file")
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	for _, l := range logs {
		if err := l.writeLTSV(f); err != nil {
			return err
		}
	}

	return err
}
