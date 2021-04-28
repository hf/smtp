package smtp

import (
	"context"
	"crypto/tls"
	"go.uber.org/zap"
	"net"
	"os"
	"sync"
)

// A SMTP server configuration.
type Config struct {
	// SMTP service's domain. This should be the same domain advertised in the
	// recipient's MX records, as well as the same CN of the TLS certificate.
	// If you don't specify this the ServerName from TLS will be used, and if
	// that's not available "example.com" will be used.
	Domain string

	// Size of the buffer per connection. Avoid setting this below 538 bytes,
	// as that is the standard line length of SMTP (512 + 26 for SIZE). If
	// unspecified will use 4 pages.
	BufferSize uint

	// TLS config for the server.
	TLS *tls.Config

	// Whether this SMTP server requires STARTTLS. Does not make sense if TLS is nil.
	TLSRequired bool

	// Callback for creating a new envelope.
	NewEnvelope func(ctx context.Context, sess *Session) (Envelope, error)

	// Logger for the server. If you do not specify this NewExample() from Zap will be used.
	Logger *zap.Logger
}

// An SMTP Server, use NewServer to create one.
type Server struct {
	Config *Config

	context    context.Context
	bufferPool *sync.Pool

	wait *sync.WaitGroup
}

// Creates a new Server with the provided Config.
func NewServer(config Config) *Server {
	if nil == config.Logger {
		config.Logger = zap.NewExample()
		config.Logger.Warn("server configured without a Logger, using NewExample()")
	}

	if "" == config.Domain && nil != config.TLS {
		config.Domain = config.TLS.ServerName
	}

	if "" == config.Domain {
		config.Logger.Warn("server configured without a Domain or TLS config, using example.com")
		config.Domain = "example.com"
	}

	config.Logger = config.Logger.Named("server").With(zap.String("server", config.Domain))

	if nil == config.NewEnvelope {
		config.Logger.Panic("server configured without a NewEnvelope")
	}

	if 0 == config.BufferSize {
		config.BufferSize = uint(4 * os.Getpagesize())
	}

	if config.BufferSize < 538 {
		config.Logger.Warn("server configured with BufferSize less than 538, which is not recommended", zap.Uint("BufferSize", config.BufferSize))
	}

	return &Server{
		Config:  &config,
		context: context.Background(),
		bufferPool: &sync.Pool{
			New: func() interface{} {
				return make([]byte, config.BufferSize)
			},
		},
		wait: &sync.WaitGroup{},
	}
}

func (srv *Server) dialog(ctx context.Context, session *Session, conn net.Conn, logger *zap.Logger) {
	var reply []byte = nil
	var err error = nil
	var action sessionAction = keepSession
	var running bool = true
	var n int = 0
	var buffer []byte = srv.bufferPool.Get().([]byte)
	var fill []byte = buffer
	var readConn net.Conn = conn

	readCtx, cancel := context.WithCancel(srv.context)

	kill := func() {
		logger.Debug("killing")

		reply, err = session.kill(readCtx)
		if nil != err {
			logger.Warn("kill failed", zap.Error(err))
		}

		if nil != reply {
			readConn.Write(reply)
		}

		running = false
	}

	read := func() {
		n, err = readConn.Read(fill)

		if 0 == n && nil != err {
			logger.Debug("end-of-stream", zap.Error(err))

			running = false
		} else {
			shouldKill := false

			remaining := readLines(buffer[:len(buffer)-len(fill)+n], func(line []byte) lineControl {
				reply, action, err = session.advance(readCtx, line)
				if nil != err {
					logger.Warn("advance failed", zap.Error(err))
				}

				if nil != reply {
					_, err = readConn.Write(reply)

					if nil != err {
						shouldKill = true
						return discardLines
					}
				}

				switch action {
				case closeSession, upgradeSession:
					return discardLines
				}

				return readMoreLines
			})

			if nil != remaining {
				if len(remaining) == len(buffer) {
					// this buffer does not contain a line, kill the connection
					logger.Warn("buffer did not contain a line, check the BufferSize config", zap.Int("BufferSize", len(buffer)))
					kill()
				} else {
					copy(buffer, remaining)
					fill = buffer[len(remaining):]
				}
			} else {
				fill = buffer

				if shouldKill {
					kill()
					running = false
				} else {
					switch action {
					case upgradeSession:
						logger.Debug("upgrade action")

						tlsConn := tls.Server(readConn, srv.Config.TLS)
						readConn = tlsConn

						err = tlsConn.Handshake()
						if nil != err {
							logger.Warn("tls handshake failed", zap.Error(err))

							running = false
						}

					case closeSession:
						logger.Debug("close action")
						running = false
					}
				}
			}
		}
	}

	done := ctx.Done()

	logger.Debug("greeting")

	_, err = readConn.Write(session.greet(readCtx))

	if nil != err {
		logger.Warn("greeting failed", zap.Error(err))
	} else {
		if nil == done {
			logger.Debug("context does not support cancellation")

			for running {
				read()
			}
		} else {
			for running {
				select {
				case <-done:
					logger.Debug("context cancelled")

					kill()

				default:
					read()
				}
			}
		}
	}

	srv.bufferPool.Put(buffer)
	buffer = nil

	cancel()

	logger.Debug("closing")

	err = readConn.Close()
	if nil != err {
		logger.Warn("close failed", zap.Error(err))
	}
}

func (srv *Server) handle(ctx context.Context, conn net.Conn, sessionFn func(ctx context.Context, srv *Server, sess *Session, init bool)) {
	id := generateID()
	addr := conn.RemoteAddr().String()

	logger := srv.Config.Logger.Named("session").With(
		zap.String("id", id),
		zap.String("addr", addr),
	)

	session := &Session{
		ID:   id,
		Addr: addr,
		config: sessionConfig{
			domain:      srv.Config.Domain,
			tls:         nil != srv.Config.TLS,
			tlsRequired: srv.Config.TLSRequired,
			newEnvelope: srv.Config.NewEnvelope,
			logger:      logger,
		},
	}

	if nil != sessionFn {
		sessionFn(ctx, srv, session, true)
	}

	srv.dialog(ctx, session, conn, logger)

	if nil != sessionFn {
		sessionFn(ctx, srv, session, false)
	}

	srv.wait.Done()
}

// Accept a new SMTP connection. The sessionFn callback will be called from
// within the goroutine handling the dialog twice: at the initialization of the
// session and once the dialog has finished. Use Wait to wait all goroutines
// started with this to finish. Cancelling the context will close all dialogs
// in an orderly fashion.
func (srv *Server) Accept(ctx context.Context, conn net.Conn, sessionFn func(ctx context.Context, srv *Server, sess *Session, init bool)) {
	srv.wait.Add(1)
	go srv.handle(ctx, conn, sessionFn)
}

// Waits for all goroutines started from within Accept to finish orderly.
func (srv *Server) Wait() {
	srv.wait.Wait()
}
