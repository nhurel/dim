package manners

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

// NewListener wraps an existing listener for use with
// GracefulServer.
//
// Note that you generally don't need to use this directly as
// GracefulServer will automatically wrap any non-graceful listeners
// supplied to it.
func NewListener(l net.Listener) *GracefulListener {
	return &GracefulListener{
		listener: l,
		mutex:    &sync.RWMutex{},
		open:     true,
	}
}

// A gracefulCon wraps a normal net.Conn and tracks the last known http state.
type gracefulConn struct {
	net.Conn
	lastHTTPState http.ConnState
	// protected tells whether the connection is going to defer server shutdown
	// until the current HTTP request is completed.
	protected bool
}

type gracefulAddr struct {
	net.Addr
	gconn *gracefulConn
}

func (g *gracefulConn) LocalAddr() net.Addr {
	return &gracefulAddr{g.Conn.LocalAddr(), g}
}

// retrieveGracefulConn retrieves a concrete gracefulConn instance from an
// interface value that can either refer to it directly or refer to a tls.Conn
// instance wrapping around a gracefulConn one.
func retrieveGracefulConn(conn net.Conn) *gracefulConn {
	return conn.LocalAddr().(*gracefulAddr).gconn
}

// A GracefulListener differs from a standard net.Listener in one way: if
// Accept() is called after it is gracefully closed, it returns a
// listenerAlreadyClosed error. The GracefulServer will ignore this error.
type GracefulListener struct {
	listener net.Listener
	open     bool
	mutex    *sync.RWMutex
}

func (l *GracefulListener) isClosed() bool {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return !l.open
}

func (l *GracefulListener) Addr() net.Addr {
	return l.listener.Addr()
}

// Accept implements the Accept method in the Listener interface.
func (l *GracefulListener) Accept() (net.Conn, error) {
	conn, err := l.listener.Accept()
	if err != nil {
		if l.isClosed() {
			err = listenerAlreadyClosed{err}
		}
		return nil, err
	}

	// don't wrap connection if it's tls so we won't break
	// http server internal logic that relies on the type
	if _, ok := conn.(*tls.Conn); ok {
		return conn, nil
	}
	return &gracefulConn{Conn: conn}, nil
}

// Close tells the wrapped listener to stop listening.  It is idempotent.
func (l *GracefulListener) Close() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if !l.open {
		return nil
	}
	l.open = false
	return l.listener.Close()
}

func (l *GracefulListener) GetFile() (*os.File, error) {
	return getListenerFile(l.listener)
}

func (l *GracefulListener) Clone() (net.Listener, error) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if !l.open {
		return nil, fmt.Errorf("listener is already closed")
	}

	file, err := l.GetFile()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fl, err := net.FileListener(file)
	if nil != err {
		return nil, err
	}
	return fl, nil
}

// A listener implements a network listener (net.Listener) for TLS connections.
// direct lift from crypto/tls.go
type TLSListener struct {
	net.Listener
	config *tls.Config
}

// Accept waits for and returns the next incoming TLS connection.
// The returned connection c is a *tls.Conn.
func (l *TLSListener) Accept() (c net.Conn, err error) {
	c, err = l.Listener.Accept()
	if err != nil {
		return
	}
	c = tls.Server(&gracefulConn{Conn: c}, l.config)
	return
}

// NewListener creates a Listener which accepts connections from an inner
// Listener and wraps each connection with Server.
// The configuration config must be non-nil and must have
// at least one certificate.
func NewTLSListener(inner net.Listener, config *tls.Config) net.Listener {
	l := new(TLSListener)
	l.Listener = inner
	l.config = config
	return l
}

type listenerAlreadyClosed struct {
	error
}

// TCPKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
//
// direct lift from net/http/server.go
type TCPKeepAliveListener struct {
	*net.TCPListener
}

func (ln TCPKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

func getListenerFile(listener net.Listener) (*os.File, error) {
	switch t := listener.(type) {
	case *net.TCPListener:
		return t.File()
	case *net.UnixListener:
		return t.File()
	case TCPKeepAliveListener:
		return t.TCPListener.File()
	case *TLSListener:
		return getListenerFile(t.Listener)
	}
	return nil, fmt.Errorf("Unsupported listener: %T", listener)
}
