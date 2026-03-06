package web

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
)

// OneTimeServer records the details needed to start a web-server that will
// server a page once and then stop
type OneTimeServer struct {
	hdlr     http.Handler
	listener net.Listener
	svr      *http.Server
	done     chan struct{}
	svrErr   chan error
	port     int
}

// MakeOneTimeServer returns a web server ready to be started
func MakeOneTimeServer(hdlr http.Handler) (*OneTimeServer, error) {
	var err error

	s := &OneTimeServer{
		done:   make(chan struct{}),
		svrErr: make(chan error),
		hdlr:   hdlr,
	}

	s.listener, err = net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, fmt.Errorf("could not open the Listener: %w", err)
	}

	s.port = s.listener.Addr().(*net.TCPAddr).Port

	s.svr = &http.Server{
		Addr:              fmt.Sprintf(":%d", s.port),
		Handler:           s,
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       time.Second,
		WriteTimeout:      time.Second,
		IdleTimeout:       time.Second,
	}

	return s, err
}

// Start starts the web server
func (s *OneTimeServer) Start() error {
	go func() {
		fmt.Println("open your web browser with URL:")
		fmt.Printf("http://localhost:%d\n", s.port)

		err := s.svr.Serve(s.listener)
		if err != nil && err != http.ErrServerClosed {
			s.svrErr <- err

			fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		}
	}()

	// wait until the server has been contacted and the web page sent or an
	// error has been raised
	select {
	case <-s.done:
		fmt.Println("Done")
	case err := <-s.svrErr:
		fmt.Fprintf(os.Stderr, "Aborting: Server error: %v\n", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Gracefully shutdown server
	err := s.svr.Shutdown(ctx)

	return err
}

// ServeHTTP calls the supplied handler and then sends a message on the done
// channel
func (s *OneTimeServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	defer func() {
		s.done <- struct{}{}
	}()

	s.hdlr.ServeHTTP(rw, req)
}
