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
