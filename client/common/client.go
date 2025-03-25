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
    conn   net.Conn
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
    log.Infof("action: all_batches_sent | result: success | client_id: %v | total_bets: %v", c.config.ID, len(bets))
}

// readBetsFromFile lee el archivo CSV y devuelve un slice de líneas, cada línea con 5 campos: A,B,document,birthdate,number
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

// chunkBets divide un slice de apuestas en sub-slices de tamaño batchSize
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

// sendBatchAndAwaitResponse serializa el batch en un solo string con '\n' interno + '\n' final, lo envía y espera respuesta
func (c *Client) sendBatchAndAwaitResponse(batch []string) error {
    // 1) Create connection
    conn, err := net.Dial("tcp", c.config.ServerAddress)
    if err != nil {
        return fmt.Errorf("connect fail: %w", err)
    }
    defer conn.Close()

    // 2) Serialize: each bet in a line, joined by '\n', + at the end '\n' to delimit
	// Exapmple: "A,B,00000000,2000-01-01,0\nA,B,00000001,2000-01-01,1\n"
    batchData := strings.Join(batch, "\n") + "\n"
    _, err = conn.Write([]byte(batchData))
    if err != nil {
        return fmt.Errorf("write fail: %w", err)
    }

    // 3) Read response (p.ej. "success|N\n" o "fail|N\n")
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
