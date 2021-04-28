package smtp

import (
	"bytes"
)

type lineControl = int

const (
	readMoreLines lineControl = iota
	discardLines              = iota
)

func readLines(buffer []byte, cb func(line []byte) lineControl) []byte {
	chunk := buffer

	for {
		crlf := bytes.Index(chunk, []byte("\r\n"))

		if crlf < 0 {
			return chunk
		} else {
			crlf += 2
			line := chunk[:crlf]
			chunk = chunk[crlf:]

			switch cb(line) {
			case discardLines:
				return nil
			}
		}
	}
}
