package smtp

import (
	"bytes"
	tst "testing"
)

func TestReadLines(t *tst.T) {
	buffer := []byte("MAIL FROM:<someone@example.com>\r\nDATA")

	calls := 0
	leftover := readLines(buffer, func(line []byte) lineControl {
		calls += 1

		if !bytes.Equal(line, []byte("MAIL FROM:<someone@example.com>\r\n")) {
			t.Errorf("Unexpected line %q", string(line))
		}

		return readMoreLines
	})

	if !bytes.Equal(leftover, []byte("DATA")) {
		t.Errorf("Unexpected leftover %q", string(leftover))
	}

	if 1 != calls {
		t.Errorf("Unexpected number of callbacks: %v", calls)
	}
}

func TestReadLinesStopReading(t *tst.T) {
	calls := 0
	leftover := readLines([]byte("ABC\r\n"), func(line []byte) lineControl {
		calls += 1
		return discardLines
	})

	if nil != leftover {
		t.Errorf("Unexpected leftover: %q", string(leftover))
	}

	if 1 != calls {
		t.Errorf("Unexpected number of callbacks: %v", calls)
	}
}
