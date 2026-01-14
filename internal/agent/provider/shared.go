package provider

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Buffer size constants for scanner operations
const (
	// ScannerInitialBufSize is the initial buffer size for scanning output (64KB)
	ScannerInitialBufSize = 64 * 1024
	// ScannerMaxBufSize is the maximum buffer size for scanning output (1MB)
	ScannerMaxBufSize = 1024 * 1024
)

// streamOutput reads from reader and writes to both buffer and writer.
// This is shared between providers for consistent output handling.
func streamOutput(reader io.Reader, buf *strings.Builder, writer io.Writer) {
	scanner := bufio.NewScanner(reader)
	// Increase scanner buffer for long lines
	scanner.Buffer(make([]byte, ScannerInitialBufSize), ScannerMaxBufSize)

	for scanner.Scan() {
		line := scanner.Text()
		buf.WriteString(line)
		buf.WriteString("\n")
		fmt.Fprintln(writer, line)
	}
}
