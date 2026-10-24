package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type flushRecorder struct {
	*httptest.ResponseRecorder
	flushed int
}

func (r *flushRecorder) Flush() {
	r.flushed++
	if r.ResponseRecorder.Flushed {
		return
	}
	r.ResponseRecorder.Flush()
}

func TestStatusWriterPreservesFlusher(t *testing.T) {
	for name, mw := range map[string]func(http.HandlerFunc) http.HandlerFunc{
		"logger": Logger(),
		"trace":  Trace(),
	} {
		t.Run(name, func(t *testing.T) {
			rec := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
			sawFlusher := false
			handler := mw(func(w http.ResponseWriter, _ *http.Request) {
				flusher, ok := w.(http.Flusher)
				sawFlusher = ok
				if ok {
					w.WriteHeader(http.StatusOK)
					flusher.Flush()
				}
			})

			handler(rec, httptest.NewRequest(http.MethodGet, "/api/test", nil))

			require.True(t, sawFlusher, "handler writer must implement http.Flusher")
			assert.Equal(t, 1, rec.flushed, "Flush must reach the underlying writer")
		})
	}
}
