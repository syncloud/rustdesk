package auth

import (
	"context"
	"net"
	"net/http"
)

type authTransport struct {
	socket http.RoundTripper
}

func newAuthTransport(socket string) http.RoundTripper {
	return &authTransport{
		socket: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", socket)
			},
		},
	}
}

func (t *authTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	viaSocket := r.Clone(r.Context())
	viaSocket.URL.Scheme = "http"
	viaSocket.Host = r.URL.Host
	viaSocket.Header.Set("X-Forwarded-Proto", "https")
	viaSocket.Header.Set("X-Forwarded-Host", r.URL.Host)
	return t.socket.RoundTrip(viaSocket)
}
