package smtp

import (
	"bytes"
	"fmt"
	"strings"
	tst "testing"
)

func TestParseCommand(t *tst.T) {
	expected := map[string]struct {
		name     commandName
		addr     []byte
		sizeHint uint64
	}{
		"HELO": {
			name: commandHELO,
		},
		"HELO domain.com": {
			name: commandHELO,
			addr: []byte("domain.com"),
		},
		"HELO domain.com says hi": {
			name: commandHELO,
			addr: []byte("domain.com"),
		},
		"EHLO": {
			name: commandEHLO,
		},
		"EHLO domain.com": {
			name: commandEHLO,
			addr: []byte("domain.com"),
		},
		"EHLO domain.com says hi": {
			name: commandEHLO,
			addr: []byte("domain.com"),
		},
		"RCPT": {
			name: commandRCPT,
		},
		"RCPT TO:<someone@example.com>": {
			name: commandRCPT,
			addr: []byte("someone@example.com"),
		},
		"RCPT TO:someone@example.com": {
			name: commandRCPT,
		},
		"MAIL": {
			name: commandMAIL,
		},
		"MAIL FROM:<someone@example.com>": {
			name: commandMAIL,
			addr: []byte("someone@example.com"),
		},
		"MAIL FROM:someone@example.com": {
			name: commandMAIL,
		},
		"MAIL SIZE=123": {
			name:     commandMAIL,
			sizeHint: 123,
		},
		"MAIL SIZE=0": {
			name:     commandMAIL,
			sizeHint: 0,
		},
		"MAIL SIZE:123": {
			name: commandMAIL,
		},
		"MAIL FROM:<someone@example.com> SIZE=123": {
			name:     commandMAIL,
			addr:     []byte("someone@example.com"),
			sizeHint: 123,
		},
		"MAIL SIZE=123 FROM:<someone@example.com>": {
			name:     commandMAIL,
			addr:     []byte("someone@example.com"),
			sizeHint: 123,
		},
		"DATA": {
			name: commandDATA,
		},
		"RSET": {
			name: commandRSET,
		},
		"QUIT": {
			name: commandQUIT,
		},
		"EXPN": {
			name: commandEXPN,
		},
		"VRFY": {
			name: commandVRFY,
		},
		"HELP": {
			name: commandHELP,
		},
		"NOOP": {
			name: commandNOOP,
		},
		"STARTTLS": {
			name: commandSTARTTLS,
		},
	}

	for cmd, s := range expected {
		line := []byte(fmt.Sprintf("%s\r\n", strings.ToLower(cmd)))
		parsed, result := parseCommand(line)

		if parseOk != result {
			t.Errorf("Unexpected parse result for command %v: %v", cmd, result)
		} else {
			if parsed.name != s.name {
				t.Errorf("Unexpected type for command %v: %v", cmd, parsed.name)
			}

			if !bytes.Equal(parsed.addr, s.addr) {
				t.Errorf("Unexpected value for Addr %v: %v", cmd, parsed.addr)
			}

			if parsed.sizeHint != s.sizeHint {
				t.Errorf("Unexpected value for SizeHint %v: %v", cmd, parsed.sizeHint)
			}
		}
	}
}

func TestParseCommandUnknown(t *tst.T) {
	line := []byte("UNKNOWN\r\n")
	_, result := parseCommand(line)

	if parseUnrecognizedCommand != result {
		t.Errorf("Unexpected parse result for %q line: %v", line, result)
	}
}

func TestParseCommandBadFormat(t *tst.T) {
	lines := []string{
		"UNKNOWN",
		"UNKNOWN\r\r\n",
		" UNKNOWN\r\n",
		"123UNKNOWN\r\n",
	}

	for _, line := range lines {
		_, result := parseCommand([]byte(line))

		if parseBadFormat != result {
			t.Errorf("Unexpected parse result for %q line: %v", line, result)
		}
	}
}
