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

var log = logging.MustGetLogger("log")

// ClientConfig includes batch.maxAmount from config.yaml, in addition to the legacy fields
type ClientConfig struct {
    ID            string
    ServerAddress string
    LoopAmount    int
    LoopPeriod    time.Duration
    MaxBatch      int           // batch.maxAmount from config.yaml
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

// StartClientBatch reads the file "agency-{ID}.csv", chunks the bets, and sends them to the server.
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

    // 2) Read CSV: "agency-{ID}.csv"
    filename := fmt.Sprintf("/app/.data/agency-%s.csv", c.config.ID)
    bets, err := c.readBetsFromFile(filename)
    if err != nil {
        log.Errorf("action: read_file | result: fail | error: %v", err)
        return
    }
    if len(bets) == 0 {
        log.Infof("action: no_bets_found | result: success | client_id: %v", c.config.ID)
        return
    }

    // 3) Group in batches of size c.config.MaxBatch
    batches := chunkBets(bets, c.config.MaxBatch)

    // 4) For each batch, send and await response
    for _, batch := range batches {
        if err := c.sendBatchAndAwaitResponse(batch); err != nil {
            log.Errorf("action: send_batch | result: fail | error: %v", err)
            return
        }
    }
    log.Infof("action: all_batches_sent | result: success | client_id: %v | total_bets: %v",
        c.config.ID, len(bets))

    // 5) Notify the server that this agency finished sending bets
    if err := c.NotifyFinished(); err != nil {
        return
    }

    // 6) Query the winners (if the server already did the draw, we get the results)
    if err := c.QueryWinners(); err != nil {
        return
    }

    // 7) After everything, log "exit" so the tests can detect we ended properly
    log.Infof("action: exit | result: success | client_id: %s", c.config.ID)
}

// readBetsFromFile loads each line from the CSV and returns them as a slice of strings
func (c *Client) readBetsFromFile(filename string) ([]string, error) {
    file, err := os.Open(filename)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var bets []string
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line != "" {
            bets = append(bets, line)
        }
    }
    return bets, scanner.Err()
}

// chunkBets splits a slice of bets into smaller chunks of batchSize
func chunkBets(bets []string, batchSize int) [][]string {
    var chunks [][]string
    for i := 0; i < len(bets); i += batchSize {
        end := i + batchSize
        if end > len(bets) {
            end = len(bets)
        }
        chunks = append(chunks, bets[i:end])
    }
    return chunks
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
    // Example:
    //   agency_ID|2
    //   Name,Surname,12345678,2000-01-01,100
    //   Name2,Surname2,23456789,2000-01-01,101
    //   ...
    var sb strings.Builder
    sb.WriteString(fmt.Sprintf("agency_ID|%s\n", c.config.ID))
    for _, line := range batch {
        sb.WriteString(line)
        sb.WriteString("\n")
    }

    // 3) Send it
    if _, err := conn.Write([]byte(sb.String())); err != nil {
        return fmt.Errorf("write fail: %w", err)
    }

    // 4) Read response (e.g. "success|10\n" or "fail|0\n")
    response, err := bufio.NewReader(conn).ReadString('\n')
    if err != nil && err != io.EOF {
        return fmt.Errorf("read fail: %w", err)
    }

    // 4) Parse response
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

    // 5) Log according to response
    if status == "success" {
        log.Infof("action: apuesta_enviada | result: success | batch_size: %s", countStr)
    } else {
        log.Errorf("action: apuesta_enviada | result: fail | batch_size: %s", countStr)
    }
    return nil
}

// NotifyFinished sends "notify_finished|<agency>" to tell the server we are done sending bets
func (c *Client) NotifyFinished() error {
    conn, err := net.Dial("tcp", c.config.ServerAddress)
    if err != nil {
        log.Criticalf("action: notify_connect | result: fail | error: %v", err)
        return err
    }
    defer conn.Close()
    message := fmt.Sprintf("notify_finished|%s\n", c.config.ID)
    _, err = conn.Write([]byte(message))
    if err != nil {
        log.Errorf("action: notify_send | result: fail | error: %v", err)
        return err
    }

    response, err := bufio.NewReader(conn).ReadString('\n')
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

// QueryWinners sends a request "query_winners|<agency>" and prints the results or fail
func (c *Client) QueryWinners() error {
    conn, err := net.Dial("tcp", c.config.ServerAddress)
    if err != nil {
        log.Criticalf("action: query_connect | result: fail | error: %v", err)
        return err
    }
    defer conn.Close()
    message := fmt.Sprintf("query_winners|%s\n", c.config.ID)
    _, err = conn.Write([]byte(message))
    if err != nil {
        log.Errorf("action: query_send | result: fail | error: %v", err)
        return err
    }

    reader := bufio.NewReader(conn)

	// Read header: ok|N
    header, err := reader.ReadString('\n')
    if err != nil {
        log.Errorf("action: query_receive_header | result: fail | error: %v", err)
        return err
    }
    header = strings.TrimSpace(header)

    // If server not done, it might say "fail-sorteo_no_listo"
    if strings.HasPrefix(header, "fail|") {
        log.Errorf("action: consulta_ganadores | result: fail | reason: %s", header)
        return nil
    }

    // Otherwise, expect "ok|<N>"
    parts := strings.Split(header, "|")
    if len(parts) != 2 || parts[0] != "ok" {
        return fmt.Errorf("invalid header response: %s", header)
    }
    count, err := strconv.Atoi(parts[1])
    if err != nil {
        return fmt.Errorf("invalid count: %s", parts[1])
    }
	
    for i := 0; i < count; i++ {
        // read each document line
        _, docErr := reader.ReadString('\n')
        if docErr != nil {
            log.Errorf("failed reading winner %d: %v", i+1, docErr)
            return docErr
        }
    }

    log.Infof("action: consulta_ganadores | result: success | cant_ganadores: %d", count)
    return nil
}
