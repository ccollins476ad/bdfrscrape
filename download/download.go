package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

// GetBody performs an http GET with url=u using the suppplied client and
// header.
func GetBody(ctx context.Context, hc *http.Client, u string, header http.Header) (io.ReadCloser, error) {
	log.Debugf("get: %s", u)

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	for k, vs := range header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}

	rsp, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	if rsp.StatusCode < 200 || rsp.StatusCode >= 300 {
		return nil, fmt.Errorf("error status: %s", rsp.Status)
	}

	return rsp.Body, nil
}

// Get calls GetBody(), then reads the full response and returns the result.
func Get(ctx context.Context, hc *http.Client, u string, header http.Header) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	body, err := GetBody(ctx, hc, u, header)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	return io.ReadAll(NewContextReader(ctx, body))
}
