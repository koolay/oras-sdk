package option

import (
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/exp/slog"
)

// requestCount records the number of logged request-response pairs and will
// be used as the unique id for the next pair.

// Transport is an http.RoundTripper that keeps track of the in-flight
// request and add hooks to report HTTP tracing events.
type Transport struct {
	http.RoundTripper
}

// newTransport creates and returns a new instance of Transport
func newTransport(base http.RoundTripper) *Transport {
	return &Transport{
		RoundTripper: base,
	}
}

// RoundTrip calls base roundtrip while keeping track of the current request.
func (t *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	slog.Debug("request", "url", req.URL, "method", req.Method)

	// log the response
	resp, err = t.RoundTripper.RoundTrip(req)
	if err != nil {
		slog.Error("Error in getting response", "error", err)
	} else if resp == nil {
		slog.Error("No response obtained for request", "url", req.URL)
	} else {
		slog.Debug("Response", "status", resp.Status)
	}
	return resp, err
}

// logHeader prints out the provided header keys and values, with auth header
// scrubbed.
func logHeader(header http.Header) string {
	if len(header) > 0 {
		headers := []string{}
		for k, v := range header {
			if strings.EqualFold(k, "Authorization") {
				v = []string{"*****"}
			}
			headers = append(headers, fmt.Sprintf("   %q: %q", k, strings.Join(v, ", ")))
		}
		return strings.Join(headers, "\n")
	}
	return "   Empty header"
}
