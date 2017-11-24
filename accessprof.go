package accessprof

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
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
		accessedAtLabel, l.AccessedAt.Format(time.RFC3339Nano),
	)
	return errors.Wrap(err, "failed to write accesslog as ltsv")
}

type AccessProf struct {
	mu         sync.Mutex
	accessLogs []*AccessLog
	// LogFile is a filepath of the log file. (if empty, accessprof holds all logs on memory)
	LogFile        string
	FlushThreshold int
	flushMu        sync.Mutex
}

func (a *AccessProf) Wrap(h http.Handler, reportPath string) *Handler {
	return &Handler{Handler: h, ReportPath: reportPath, AccessProf: a}
}

func (a *AccessProf) Count() int {
	a.mu.Lock()
	n := len(a.accessLogs)
	a.mu.Unlock()
	return n
}

func (a *AccessProf) Report(aggregates []*regexp.Regexp) *Report {
	a.flushLogs()
	logs, err := a.LoadAccessLogs()
	if err != nil {
		panic(err)
	}

	a.mu.Lock()
	logs = append(logs, a.accessLogs...)
	a.mu.Unlock()

	var (
		segs  []*ReportSegment
		since time.Time
	)

	for _, l := range logs {
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

func (a *AccessProf) Reset() {
	a.mu.Lock()
	a.accessLogs = a.accessLogs[:0]
	a.mu.Unlock()
}

func (a *AccessProf) flushLogs() (err error) {
	if a.LogFile == "" {
		return nil
	}
	a.flushMu.Lock()
	defer a.flushMu.Unlock()

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

func (a *AccessProf) LoadAccessLogs() ([]*AccessLog, error) {
	if a.LogFile == "" {
		return nil, nil
	}
	a.flushMu.Lock()
	defer a.flushMu.Unlock()
	var logs []*AccessLog
	f, err := os.Open(a.LogFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open access logs")
	}
	defer f.Close()
	r := bufio.NewScanner(f)
	line := 0
	for r.Scan() {
		line++
		log, err := parseLTSV(r.Text())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse log at line %d", line)
		}
		logs = append(logs, log)
	}
	if err := r.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to read access logs")
	}
	return logs, nil
}

func parseLTSV(s string) (*AccessLog, error) {
	columns := strings.Split(s, "\t")
	table := map[string]string{}
	for _, column := range columns {
		ss := strings.SplitN(column, ":", 2)
		table[ss[0]] = ss[1]
	}
	l := new(AccessLog)
	if s, ok := table[methodLabel]; ok {
		l.Method = s
	} else {
		return nil, errors.New("missing method label")
	}
	if s, ok := table[pathLabel]; ok {
		l.Path = s
	} else {
		return nil, errors.New("missing path label")
	}
	if s, ok := table[statusLabel]; ok {
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse status code")
		}
		l.Status = n
	} else {
		return nil, errors.New("missing status label")
	}
	if s, ok := table[responseBodySizeLabel]; ok {
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse response body size")
		}
		l.ResponseBodySize = n
	} else {
		return nil, errors.New("missing response body size label")
	}
	if s, ok := table[responseTimeLabel]; ok {
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse response time")
		}
		l.ResponseTime = time.Nanosecond * time.Duration(n)
	} else {
		return nil, errors.New("missing response time label")
	}
	if s, ok := table[accessedAtLabel]; ok {
		t, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse accessed_at")
		}
		l.AccessedAt = t
	} else {
		return nil, errors.New("missing accessed_at label")
	}
	return l, nil
}

type Handler struct {
	// Handler is the base handler to wrap
	Handler http.Handler
	// ReportPath is a path of HTML reporting endpoint (ignored if empty)
	ReportPath string
	*AccessProf
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
			re, err := regexp.Compile("^" + agg + "$")
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
