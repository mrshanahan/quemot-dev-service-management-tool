package utils

import (
	"bytes"
	"fmt"
	"os"
)

func PrintErrln(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}

func PrintErrf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
}

// dropCR & the definition of ScanLinesOrUntil were basically copied verbatim
// from the golang source: https://github.com/golang/go/blob/master/src/bufio/scan.go#L341-L369

// dropCR drops a terminal \r from the data.
func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

func ScanUntil(bs ...byte) func([]byte, bool) (int, []byte, error) {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}

		for _, b := range bs {
			if i := bytes.IndexByte(data, b); i >= 0 {
				// We have a full newline-terminated line.
				// MRS: OR terminated by whatever was passed in!
				return i + 1, dropCR(data[0:i]), nil
			}
		}
		// If we're at EOF, we have a final, non-terminated line. Return it.
		if atEOF {
			return len(data), dropCR(data), nil
		}
		// Request more data.
		return 0, nil, nil
	}
}
