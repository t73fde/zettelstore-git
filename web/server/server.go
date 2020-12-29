//-----------------------------------------------------------------------------
// Copyright (c) 2020 Detlef Stern
//
// This file is part of zettelstore.
//
// Zettelstore is licensed under the latest version of the EUPL (European Union
// Public License). Please see file LICENSE.txt for your rights and obligations
// under this license.
//-----------------------------------------------------------------------------

// Package server provides a web server.
package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Server timeout values
const (
	shutdownTimeout = 5 * time.Second
	readTimeout     = 5 * time.Second
	writeTimeout    = 10 * time.Second
	idleTimeout     = 120 * time.Second
)

// Server is a HTTP server.
type Server struct {
	*http.Server
	waitShutdown chan struct{}
}

// New creates a new HTTP server object.
func New(addr string, handler http.Handler) *Server {
	if addr == "" {
		addr = ":http"
	}
	srv := &Server{
		Server: &http.Server{
			Addr:    addr,
			Handler: handler,

			// See: https://blog.cloudflare.com/exposing-go-on-the-internet/
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
		},
		waitShutdown: make(chan struct{}),
	}
	return srv
}

// Run starts the web server and wait for its completion.
func (srv *Server) Run() error {
	waitInterrupt := make(chan os.Signal)
	waitError := make(chan error)
	signal.Notify(waitInterrupt, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case <-waitInterrupt:
		case <-srv.waitShutdown:
		}
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		log.Println("Stopping Zettelstore...")
		if err := srv.Shutdown(ctx); err != nil {
			waitError <- err
			return
		}
		waitError <- nil
	}()

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return <-waitError
}

// Stop the web server.
func (srv *Server) Stop() {
	close(srv.waitShutdown)
}
