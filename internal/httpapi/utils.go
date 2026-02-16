package httpapi

import (
	"io"
	"net/http"
)

func readAll(r *http.Request, limit int64) ([]byte, error) {
	body := http.MaxBytesReader(nil, r.Body, limit)
	defer body.Close()
	return io.ReadAll(body)
}
