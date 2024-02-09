package auth

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
)

// codeResponse represents the code received by the local server's callback handler.
type codeResponse struct {
	Code  string
	State string
}

// bindLocalServer initializes a LocalServer that will listen on a randomly available TCP port.
func bindLocalServer() (*localServer, error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:10280")
	if err != nil {
		return nil, err
	}

	return &localServer{
		listener:     listener,
		resultChan:   make(chan codeResponse, 1),
		callbackPath: "/oauth/callback",
	}, nil
}

type localServer struct {
	WriteSuccessHTML func(w io.Writer)

	resultChan   chan (codeResponse)
	listener     net.Listener
	callbackPath string
}

func (s *localServer) Port() int {
	return s.listener.Addr().(*net.TCPAddr).Port
}

func (s *localServer) Close() error {
	return s.listener.Close()
}

func (s *localServer) Serve() error {
	return http.Serve(s.listener, s)
}

func (s *localServer) URL() string {
	return fmt.Sprintf("http://%s%s", s.listener.Addr().String(), s.callbackPath)
}

func (s *localServer) WaitForCode(ctx context.Context) (codeResponse, error) {
	select {
	case <-ctx.Done():
		return codeResponse{}, ctx.Err()
	case code := <-s.resultChan:
		return code, nil
	}
}

// ServeHTTP implements http.Handler.
func (s *localServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != s.callbackPath {
		w.WriteHeader(404)
		return
	}
	defer func() {
		_ = s.Close()
	}()

	params := r.URL.Query()
	s.resultChan <- codeResponse{
		Code:  params.Get("code"),
		State: params.Get("state"),
	}

	w.Header().Add("content-type", "text/html")
	if s.WriteSuccessHTML != nil {
		s.WriteSuccessHTML(w)
	} else {
		defaultSuccessHTML(w)
	}
}

func defaultSuccessHTML(w io.Writer) {
	fmt.Fprintf(w, "<p>You may now close this page and return to the client app.</p>")
}
