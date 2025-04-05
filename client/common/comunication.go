package common

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/op/go-logging"
)

const (
	MaxRetries = 3
	WaitTime   = 1 * time.Second
)

var comunicationLog = logging.MustGetLogger("log")

// writeFull ensures that the entire data slice is written to the connection.
// It loops until all bytes have been sent or an error occurs.
func writeFull(conn net.Conn, data []byte) error {
	totalWritten := 0
	for totalWritten < len(data) {
		n, err := conn.Write(data[totalWritten:])
		if err != nil {
			return err
		}
		totalWritten += n
	}
	return nil
}

// readResponseWithRetry attempts to read a line from the buffered reader with retries.
// This makes the reading process persistent in case of transient errors.
func readResponseWithRetry(reader *bufio.Reader) (string, error) {
	var response string
	var err error
	for attempt := 1; attempt <= MaxRetries; attempt++ {
		response, err = reader.ReadString('\n')
		if err == nil || err == io.EOF {
			return response, nil
		}
		comunicationLog.Errorf("action: read_response_retry | attempt: %d | error: %v", attempt, err)
		time.Sleep(WaitTime)
	}
	return response, err
}

// dialWithRetry tries to establish a connection with retries.
func dialWithRetry(address string) (net.Conn, error) {
	var conn net.Conn
	var err error
	for attempt := 1; attempt <= MaxRetries; attempt++ {
		conn, err = net.Dial("tcp", address)
		if err == nil {
			return conn, nil
		}
        comunicationLog.Errorf("action: dial_retry | result: in_progress | attempt: %d | error: %v", attempt, err)
		time.Sleep(WaitTime)
	}
	return nil, fmt.Errorf("failed to dial after %d attempts: %w", MaxRetries, err)
}

// sendMessage builds a message with a length header and sends it over the connection,
// then reads the response using persistent read logic.
func sendMessage(conn net.Conn, message string) (string, error) {
	data := []byte(message)
	header := fmt.Sprintf("%d;", len(data))
	fullMessage := []byte(header)
	fullMessage = append(fullMessage, data...)
	if err := writeFull(conn, fullMessage); err != nil {
		return "", err
	}
	reader := bufio.NewReader(conn)
	response, err := readResponseWithRetry(reader)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(response), nil
}
