package accessprof

import (
	"net/http"
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

var defaultAccessProf AccessProf

func Wrap(h http.Handler) http.Handler {
	return defaultAccessProf.Wrap(h)
}
