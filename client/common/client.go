package common

import (
    "bufio"
    "fmt"
    "io"
    "net"
    "os"
    "os/signal"
    "strconv"
    "strings"
    "syscall"
    "time"

    "github.com/op/go-logging"
)

const (
    MaxRetries = 3
    WaitTime   = 1 * time.Second
)

var log = logging.MustGetLogger("log")

// ClientConfig includes batch.maxAmount from config.yaml, in addition to the legacy fields
type ClientConfig struct {
    ID            string
    ServerAddress string
    LoopAmount    int
    LoopPeriod    time.Duration
    MaxBatch      int // batch.maxAmount from config.yaml
}

// Client handles reading bets from a CSV file and sending them in batches
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
        log.Infof("action: exit | result: success | client_id: %v | message: SIGTERM received", c.config.ID)
        return
    default:
        // no SIGTERM => proceed
    }

    // 2) 2) Read CSV: " agency- {ID} . csv" and send the CSV data in batches
    filename := fmt.Sprintf("/app/.data/agency-%s.csv", c.config.ID)
    total, err := c.sendBetsByChunks(filename)
    if err != nil {
        log.Errorf("action: send_chunks | result: fail | error: %v", err)
        return
    }
    if total == 0 {
        // If the file is empty or has no valid bets
        log.Infof("action: no_bets_found | result: success | client_id: %v", c.config.ID)
        return
    }

    // 3) Notify the server that this agency finished sending bets
    if err := c.NotifyFinished(); err != nil {
        return
    }

    // 4) Query the winners (if the server already did the draw, we get the results)
    if err := c.QueryWinners(); err != nil {
        return
    }

    time.Sleep(500 * time.Millisecond)

    // 5) After everything, log "exit" so the tests can detect we ended properly
    log.Infof("action: exit | result: success | client_id: %s", c.config.ID)
}

// sendBetsByChunks opens the CSV file and reads it line by line.
// Whenever the batch size c.config.MaxBatch is reached, it sends the batch
// to the server using sendBatchAndAwaitResponse(...). Then it clears the in-memory
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

        // If the batch is full, send it to the server
        if len(batch) == batchSize {
            if err := c.sendBatchWithRetry(batch); err != nil {
                return total, err
            }
            total += len(batch)
            // Reuse the slice without reallocating
            batch = batch[:0]
        }
    }

    // Send the last partial batch (if any)
    if len(batch) > 0 {
        if err := c.sendBatchWithRetry(batch); err != nil {
            return total, err
        }
        total += len(batch)
    }

    // Check for any error that happened during scanning
    if err := scanner.Err(); err != nil {
        return total, err
    }

    log.Infof("action: all_batches_sent | result: success | client_id: %v | total_bets: %v",
        c.config.ID, total)
    return total, nil
}

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

// sendBatchAndAwaitResponse sends a batch in one shot, then reads "success|N" or "fail|N"
func (c *Client) sendBatchAndAwaitResponse(batch []string) error {
    // 1) Create connection
    conn, err := net.Dial("tcp", c.config.ServerAddress)
    if err != nil {
        return fmt.Errorf("connect fail: %w", err)
    }
    defer conn.Close()

    // 2) Build data: first line "agency_ID|<ID>" + each bet on its own line + trailing "\n"
    var sb strings.Builder
    sb.WriteString(fmt.Sprintf("agency_ID|%s\n", c.config.ID))
    for _, line := range batch {
        sb.WriteString(line)
        sb.WriteString("\n")
    }
    messageBody := sb.String()
    data := []byte(messageBody)

    // Prepend header with the length of messageBody and a semicolon delimiter
    header := fmt.Sprintf("%d;", len(data))
    fullMessage := []byte(header)
    fullMessage = append(fullMessage, data...)

	// 3) Send fullMessage, ensuring all bytes are written.
	if err := writeFull(conn, fullMessage); err != nil {
		return fmt.Errorf("write fail: %w", err)
	}

    // 4) Read response using persistent read with retry (see readResponseWithRetry)
    reader := bufio.NewReader(conn)
    response, err := readResponseWithRetry(reader)
    if err != nil {
        return fmt.Errorf("read fail: %w", err)
    }

    // 5) Parse response
    response = strings.TrimSpace(response)
    parts := strings.Split(response, "|")
    if len(parts) != 2 {
        return fmt.Errorf("invalid server response: %s", response)
    }
    status := parts[0]
    countStr := parts[1]
    _, convErr := strconv.Atoi(countStr)
    if convErr != nil {
        return fmt.Errorf("invalid count in server response: %s", response)
    }

    // 6) Log according to response
    if status == "success" {
        log.Infof("action: apuesta_enviada | result: success | batch_size: %s", countStr)
    } else {
        log.Errorf("action: apuesta_enviada | result: fail | batch_size: %s", countStr)
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
        log.Errorf("action: read_response_retry | attempt: %d | error: %v", attempt, err)
        time.Sleep(WaitTime)
    }
    return response, err
}

// sendBatchWithRetry attempts to send a batch with retries in case of failure.
// This function wraps sendBatchAndAwaitResponse, retrying up to MaxRetries times with a delay of WaitTime between attempts.
func (c *Client) sendBatchWithRetry(batch []string) error {
    var err error
    for attempt := 1; attempt <= MaxRetries; attempt++ {
        err = c.sendBatchAndAwaitResponse(batch)
        if err == nil {
            // Successfully sent the batch, return nil error.
            return nil
        }
        log.Errorf("action: send_batch_retry | attempt: %d | result: fail | error: %v", attempt, err)
        time.Sleep(WaitTime)
    }
    // After MaxRetries attempts, return the last error encountered.
    return fmt.Errorf("failed to send batch after %d attempts: %w", MaxRetries, err)
}

// NotifyFinished sends "notify_finished|<agency>" to tell the server we are done sending bets,
// with persistent send/receive logic.
func (c *Client) NotifyFinished() error {
    var err error
    var conn net.Conn
    // Retry connection
    for attempt := 1; attempt <= MaxRetries; attempt++ {
        conn, err = net.Dial("tcp", c.config.ServerAddress)
        if err == nil {
            break
        }
        log.Errorf("action: notify_connect_retry | attempt: %d | error: %v", attempt, err)
        time.Sleep(WaitTime)
    }
    if err != nil {
        log.Criticalf("action: notify_connect | result: fail | error: %v", err)
        return err
    }
    defer conn.Close()

    message := fmt.Sprintf("notify_finished|%s\n", c.config.ID)
    data := []byte(message)

    // Prepend header with the message length and delimiter
    header := fmt.Sprintf("%d;", len(data))
    fullMessage := []byte(header)
    fullMessage = append(fullMessage, data...)

    if err := writeFull(conn, fullMessage); err != nil {
        log.Errorf("action: notify_send | result: fail | error: %v", err)
        return err
    }

    // Use persistent read for the response
    response, err := readResponseWithRetry(bufio.NewReader(conn))
    if err != nil {
        log.Errorf("action: notify_receive | result: fail | error: %v", err)
        return err
    }
    response = strings.TrimSpace(response)
    if response != "ack_notify" {
        log.Errorf("action: notify | result: fail | unexpected response: %s", response)
        return fmt.Errorf("unexpected response: %s", response)
    }

    log.Infof("action: notify | result: success | client_id: %s", c.config.ID)
    return nil
}

// QueryWinners retries several times until the draw (sorteo) is ready, using persistent
// send and receive logic for the query message.
func (c *Client) QueryWinners() error {
    maxRetries := 30
    wait := 1 * time.Second

    for i := 0; i < maxRetries; i++ {
        // Retry connection
        var conn net.Conn
        var err error
        for attempt := 1; attempt <= MaxRetries; attempt++ {
            conn, err = net.Dial("tcp", c.config.ServerAddress)
            if err == nil {
                break
            }
            log.Errorf("action: query_connect_retry | attempt: %d | error: %v", attempt, err)
            time.Sleep(WaitTime)
        }
        if err != nil {
            log.Criticalf("action: query_connect | result: fail | error: %v", err)
            return err
        }

        // Send query message with persistent write
        message := fmt.Sprintf("query_winners|%s\n", c.config.ID)
        data := []byte(message)
        header := fmt.Sprintf("%d;", len(data)) // Prepend header with the message length and delimiter
        fullMessage := []byte(header)
        fullMessage = append(fullMessage, data...)

        if err := writeFull(conn, fullMessage); err != nil {
			log.Errorf("action: query_send | result: fail | error: %v", err)
			conn.Close()
			return err
		}

        // 3) Read the server's response using persistence logic
        reader := bufio.NewReader(conn)
        headerResponse, err := readResponseWithRetry(reader)
        if err != nil && err != io.EOF {
            log.Errorf("action: query_receive_header | result: fail | error: %v", err)
            conn.Close()
            return err
        }
        headerResponse = strings.TrimSpace(headerResponse)

        // 4) Check if the server indicates that the draw is not ready yet
        if strings.HasPrefix(headerResponse, "in_progress-sorteo_no_listo") {
            log.Infof("action: consulta_ganadores | result: in_progress | reason: %s. Reintentando...", headerResponse)
            conn.Close()
            time.Sleep(wait)
            continue
        }

        // 5) If the server responded with a "fail", end the attempt
        if strings.HasPrefix(headerResponse, "fail-") {
            log.Errorf("action: consulta_ganadores | result: fail | reason: %s", headerResponse)
            conn.Close()
            return nil
        }

        // 6) If we're here, we expect "ok|<N>"
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

        // 7) Read the winner documents
        for j := 0; j < count; j++ {
            line, err := reader.ReadString('\n')
            if err != nil {
                log.Errorf("failed reading winner %d: %v", j+1, err)
                conn.Close()
                return err
            }
            line = strings.TrimSpace(line)
            log.Infof("winner document: %s", line)
        }

        log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %d", count)
        conn.Close()
        return nil
    }

    // If we reach this point, we exceeded the maximum number of retries
    return fmt.Errorf("exceeded maxRetries waiting for the draw (sorteo) to be ready")
}

