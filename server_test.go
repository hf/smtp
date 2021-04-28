package smtp

import (
	"bytes"
	"context"
	"go.uber.org/zap"
	"strings"
	tst "testing"
)

func TestServer(t *tst.T) {
	conn := &testConn{
		remote: &testAddr{
			network: "tcp",
			address: "127.0.0.2:2938",
		},
		local: &testAddr{
			network: "tcp",
			address: "127.0.0.1:25",
		},
		reader: bytes.NewBuffer(make([]byte, 0, 1024)),
		writer: bytes.NewBuffer(make([]byte, 0, 1024)),
	}

	envelopes := make([]*testEnvelope, 0, 10)

	server := NewServer(Config{
		Domain:     "example.com",
		BufferSize: 536,
		Logger:     zap.NewExample(),
		NewEnvelope: func(ctx context.Context, sess *Session) (Envelope, error) {
			domain := sess.Domain()

			if "domain.com" != string(domain) {
				t.Errorf("Unexpected session domain: %q", sess.Domain())
			}

			envelope := &testEnvelope{}
			envelopes = append(envelopes, envelope)

			return envelope, nil
		},
	})

	conn.reader.Write([]byte(strings.Join([]string{
		"1234",
		"UNKNOWN",
		"EHLO",
		"HELO",
		"EHLO domain.com",
		"HELO domain.com",
		"EHLO domain.com",
		"MAIL FROM:<someone@domain.com>",
		"RCPT TO:<someone@example.com>",
		"DATA",
		"hello",
		".",
		"MAIL FROM:<someone@domain.com>",
		"RCPT TO:<someone@example.com>",
		"RCPT TO:<somebody@example.com>",
		"DATA",
		"hello",
		"..",
		".",
		"MAIL FROM:<someone@domain.com>",
		"RCPT TO:<someone@example.com>",
		"RSET",
		"MAIL FROM:<someone@domain.com>",
		"DATA",
		"RSET",
		"RSET",
		"EXPN",
		"VRFY",
		"NOOP",
		"HELP",
		"STARTTLS",
		"QUIT",
		"",
	}, "\r\n")))

	ctx, cancel := context.WithCancel(context.Background())

	server.Accept(ctx, conn, nil)
	server.Wait()

	cancel()

	expected := strings.Join([]string{
		"220 example.com Service ready",
		"500 Syntax error, command unrecognized",
		"500 Syntax error, command unrecognized",
		"500 Syntax error, command unrecognized",
		"500 Syntax error, command unrecognized",
		"250-example.com greetings",
		"250-8BITMIME",
		"250 SIZE",
		"250 example.com greetings",
		"250-example.com greetings",
		"250-8BITMIME",
		"250 SIZE",
		"250 Requested mail action okay, completed",
		"250 Requested mail action okay, completed",
		"354 Start mail input; end with <CRLF>.<CRLF>",
		"250 Requested mail action okay, completed",
		"250 Requested mail action okay, completed",
		"250 Requested mail action okay, completed",
		"250 Requested mail action okay, completed",
		"354 Start mail input; end with <CRLF>.<CRLF>",
		"250 Requested mail action okay, completed",
		"250 Requested mail action okay, completed",
		"250 Requested mail action okay, completed",
		"250 Requested mail action okay, completed",
		"250 Requested mail action okay, completed",
		"503 Bad sequence of commands",
		"250 Requested mail action okay, completed",
		"250 Requested mail action okay, completed",
		"502 Command not implemented",
		"502 Command not implemented",
		"250 Requested mail action okay, completed",
		"502 Command not implemented",
		"502 Command not implemented",
		"221 example.com Service closing transmission channel",
		"",
	}, "\r\n")

	result := string(conn.writer.Bytes())

	if expected != result {
		t.Errorf("Unexpected output: %v", result)
	}
}
