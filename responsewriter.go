package accessprof

import "net/http"

type responseWriter struct {
	w      http.ResponseWriter
	status int
}

func (r *responseWriter) WriteHeader(n int) {
	r.status = n
	r.w.WriteHeader(n)
}

func (r *responseWriter) Header() http.Header {
	return r.w.Header()
}

func (r *responseWriter) Write(s []byte) (int, error) {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	return r.w.Write(s)
}
