package smtp

import (
	"bytes"
	tst "testing"
)

func TestReplyEHLOOk(t *tst.T) {
	examples := []struct {
		Domain     string
		Extensions string
		Expected   string
	}{
		{
			Domain:     "example.com",
			Extensions: "250 one\r\n",
			Expected:   "250-example.com greetings\r\n250 one\r\n",
		},
		{
			Domain:     "example.com",
			Extensions: "250-one\r\n250 two\r\n",
			Expected:   "250-example.com greetings\r\n250-one\r\n250 two\r\n",
		},
		{
			Domain:     "example.com",
			Extensions: "",
			Expected:   "250 example.com greetings\r\n",
		},
	}

	for _, ex := range examples {
		reply := replyEHLOOk(ex.Domain, ex.Extensions)

		if !bytes.Equal([]byte(ex.Expected), reply) {
			t.Errorf("Unexpected reply for example %q: %q", ex.Expected, reply)
		}
	}
}

func TestReplyserviceNotAvailable(t *tst.T) {
	example := []byte("421 example.com Service not available, closing transmission channel\r\n")
	reply := replyServiceNotAvailable("example.com")

	if !bytes.Equal(example, reply) {
		t.Errorf("Unexpected reply for example %q: %q", example, reply)
	}
}

func TestReplyServiceReady(t *tst.T) {
	example := []byte("220 example.com Service ready\r\n")
	reply := replyServiceReady("example.com")

	if !bytes.Equal(example, reply) {
		t.Errorf("Unexpected reply for example %q: %q", example, reply)
	}
}

func TestReplyServiceClosing(t *tst.T) {
	example := []byte("221 example.com Service closing transmission channel\r\n")
	reply := replyServiceClosing("example.com")

	if !bytes.Equal(example, reply) {
		t.Errorf("Unexpected reply for example %q: %q", example, reply)
	}
}
