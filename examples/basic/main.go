// This is an example server. You can use Netcat to communicate with it.
//
// First start the server with `go run github.com/hf/smpt/examples/basic`.
// Then start Netcat and talk to it as shown below:
//   > nc -C localhost 2525
// 220 example.com Service ready
// HELO domain.com
// 250 example.com greetings
// MAIL FROM:<hello@domain.com>
// 250 Requested mail action okay, completed
// RCPT TO:<hello@example.com>
// 250 Requested mail action okay, completed
// DATA
// 354 Start mail input; end with <CRLF>.<CRLF>
// Greetings!
// .
// 250 Requested mail action okay, completed
// QUIT
// 221 example.com Service closing transmission channel
// ^C

package main

import (
	"context"
	"github.com/hf/smtp"
	"go.uber.org/zap"
	"net"
)

func main() {
	logger := zap.NewExample()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := smtp.NewServer(smtp.Config{
		Domain: "example.com",
		NewEnvelope: func(ctx context.Context, sess *smtp.Session) (smtp.Envelope, error) {
			return &exampleEnvelope{
				logger: logger,
			}, nil
		},
		Logger: logger,
	})

	listener, err := net.Listen("tcp", ":2525")
	if nil != err {
		logger.Fatal("listen failed", zap.Error(err))
	}

	for {
		conn, err := listener.Accept()
		if nil != err {
			logger.Error("accept failed", zap.Error(err))
			break
		}

		server.Accept(ctx, conn, nil)
	}

	server.Wait()

	logger.Debug("bye")
}
