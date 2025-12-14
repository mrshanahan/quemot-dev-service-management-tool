package utils

import (
	"fmt"
	"os"
)

func PrintErrln(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}

func PrintErrf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
}
