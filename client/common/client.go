package common

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/op/go-logging"
)

var clientLog = logging.MustGetLogger("clientLog")

// ClientConfig includes batch.maxAmount from config.yaml, in addition to the legacy fields.
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
	MaxBatch      int // batch.maxAmount from config.yaml
}

// Client handles reading bets from a CSV file and sending them in batches.
type Client struct {
	config ClientConfig
}

// NewClient initializes a new client receiving the configuration as a parameter.
func NewClient(config ClientConfig) *Client {
	return &Client{
		config: config,
	}
}

// StartClientBatch reads the file "agency-{ID}.csv", processes bets in chunks, and sends them to the server.
func (c *Client) StartClientBatch() {
	// 1) Handle SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)
	select {
	case <-sigChan:
		clientLog.Infof("action: exit | result: success | client_id: %v | message: SIGTERM received", c.config.ID)
		return
	default:
		// no SIGTERM => proceed
	}

	// 2) Read CSV: "agency-{ID}.csv" and send the CSV data in batches.
	filename := fmt.Sprintf("/app/.data/agency-%s.csv", c.config.ID)
	total, err := c.sendBetsByChunks(filename)
	if err != nil {
		clientLog.Errorf("action: send_chunks | result: fail | error: %v", err)
		return
	}
	if total == 0 {
		// If the file is empty or has no valid bets.
		clientLog.Infof("action: no_bets_found | result: success | client_id: %v", c.config.ID)
		return
	}

	// 3) Notify the server that this agency finished sending bets.
	if err := c.NotifyFinished(); err != nil {
		return
	}

	// 4) Query the winners (if the server already did the draw, we get the results).
	if err := c.QueryWinners(); err != nil {
		return
	}

	time.Sleep(500 * time.Millisecond)

	// 5) After everything, log "exit" so the tests can detect we ended properly.
	clientLog.Infof("action: exit | result: success | client_id: %s", c.config.ID)
}

// sendBetsByChunks opens the CSV file and reads it line by line.
// Whenever the batch size c.config.MaxBatch is reached, it sends the batch
// to the server using sendBatchAndAwaitResponse. Then it clears the in-memory
// batch before continuing to read further lines.
func (c *Client) sendBetsByChunks(filename string) (int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var batch []string
	batchSize := c.config.MaxBatch
	total := 0 // total lines sent

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		batch = append(batch, line)

		// If the batch is full, send it to the server.
		if len(batch) == batchSize {
			if err := c.sendBatchWithRetry(batch); err != nil {
				return total, err
			}
			total += len(batch)
			// Reuse the slice without reallocating.
			batch = batch[:0]
		}
	}

	// Send the last partial batch (if any).
	if len(batch) > 0 {
		if err := c.sendBatchWithRetry(batch); err != nil {
			return total, err
		}
		total += len(batch)
	}

	// Check for any error that happened during scanning.
	if err := scanner.Err(); err != nil {
		return total, err
	}

	clientLog.Infof("action: all_batches_sent | result: success | client_id: %v | total_bets: %v",
		c.config.ID, total)
	return total, nil
}

// sendBatchAndAwaitResponse builds the batch message and sends it using the transport function sendMessage.
func (c *Client) sendBatchAndAwaitResponse(batch []string) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("agency_ID|%s\n", c.config.ID))
	for _, line := range batch {
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	messageBody := sb.String()

	conn, err := dialWithRetry(c.config.ServerAddress)
	if err != nil {
		return fmt.Errorf("connect fail: %w", err)
	}
	defer conn.Close()

	response, err := sendMessage(conn, messageBody)
	if err != nil {
		return fmt.Errorf("send fail: %w", err)
	}

	// Parse response, expecting "success|N" or "fail|N".
	parts := strings.Split(response, "|")
	if len(parts) != 2 {
		return fmt.Errorf("invalid server response: %s", response)
	}
	status := parts[0]
	countStr := parts[1]
	if _, convErr := strconv.Atoi(countStr); convErr != nil {
		return fmt.Errorf("invalid count in server response: %s", response)
	}

	if status == "success" {
		clientLog.Infof("action: apuesta_enviada | result: success | batch_size: %s", countStr)
	} else {
		clientLog.Errorf("action: apuesta_enviada | result: fail | batch_size: %s", countStr)
	}
	return nil
}

// sendBatchWithRetry attempts to send a batch with retries in case of failure.
// It wraps sendBatchAndAwaitResponse, retrying up to MaxRetries times with a delay of WaitTime between attempts.
func (c *Client) sendBatchWithRetry(batch []string) error {
	var err error
	for attempt := 1; attempt <= MaxRetries; attempt++ {
		err = c.sendBatchAndAwaitResponse(batch)
		if err == nil {
			// Successfully sent the batch.
			return nil
		}
		clientLog.Errorf("action: send_batch_retry | attempt: %d | result: fail | error: %v", attempt, err)
		time.Sleep(WaitTime)
	}
	return fmt.Errorf("failed to send batch after %d attempts: %w", MaxRetries, err)
}

// NotifyFinished sends "notify_finished|<agency>" to tell the server we are done sending bets,
// using persistent send/receive logic.
func (c *Client) NotifyFinished() error {
	conn, err := dialWithRetry(c.config.ServerAddress)
	if err != nil {
		clientLog.Criticalf("action: notify_connect | result: fail | error: %v", err)
		return err
	}
	defer conn.Close()

	message := fmt.Sprintf("notify_finished|%s\n", c.config.ID)
	response, err := sendMessage(conn, message)
	if err != nil {
		clientLog.Errorf("action: notify_receive | result: fail | error: %v", err)
		return err
	}
	if response != "ack_notify" {
		clientLog.Errorf("action: notify | result: fail | unexpected response: %s", response)
		return fmt.Errorf("unexpected response: %s", response)
	}

	clientLog.Infof("action: notify | result: success | client_id: %s", c.config.ID)
	return nil
}

// QueryWinners retries several times until the draw (sorteo) is ready,
// using persistent send/receive logic for the query message.
func (c *Client) QueryWinners() error {
	maxRetries := 30
	wait := 1 * time.Second

	for i := 0; i < maxRetries; i++ {
		conn, err := dialWithRetry(c.config.ServerAddress)
		if err != nil {
			clientLog.Criticalf("action: query_connect | result: fail | error: %v", err)
			return err
		}

		message := fmt.Sprintf("query_winners|%s\n", c.config.ID)
		// Build and send the query message.
		data := []byte(message)
		header := fmt.Sprintf("%d;", len(data))
		fullMessage := []byte(header)
		fullMessage = append(fullMessage, data...)
		if err := writeFull(conn, fullMessage); err != nil {
			clientLog.Errorf("action: query_send | result: fail | error: %v", err)
			conn.Close()
			return err
		}

		reader := bufio.NewReader(conn)
		headerResponse, err := readResponseWithRetry(reader)
		if err != nil {
			clientLog.Errorf("action: query_receive_header | result: fail | error: %v", err)
			conn.Close()
			return err
		}
		headerResponse = strings.TrimSpace(headerResponse)

		if strings.HasPrefix(headerResponse, "in_progress-sorteo_no_listo") {
			clientLog.Infof("action: consulta_ganadores | result: in_progress | reason: %s. Retrying...", headerResponse)
			conn.Close()
			time.Sleep(wait)
			continue
		}

		if strings.HasPrefix(headerResponse, "fail-") {
			clientLog.Errorf("action: consulta_ganadores | result: fail | reason: %s", headerResponse)
			conn.Close()
			return nil
		}

		parts := strings.Split(headerResponse, "|")
		if len(parts) != 2 || parts[0] != "ok" {
			conn.Close()
			return fmt.Errorf("invalid response from server: %s", headerResponse)
		}
		count, err := strconv.Atoi(parts[1])
		if err != nil {
			conn.Close()
			return fmt.Errorf("invalid count in response: %s", parts[1])
		}

		// Read the winner documents.
		for j := 0; j < count; j++ {
			line, err := reader.ReadString('\n')
			if err != nil {
				clientLog.Errorf("failed reading winner %d: %v", j+1, err)
				conn.Close()
				return err
			}
			line = strings.TrimSpace(line)
			clientLog.Infof("winner document: %s", line)
		}

		clientLog.Infof("action: consulta_ganadores | result: success | cant_ganadores: %d", count)
		conn.Close()
		return nil
	}

	return fmt.Errorf("exceeded maxRetries waiting for the draw (sorteo) to be ready")
}
