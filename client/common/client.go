package common

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("log")

// ClientConfig Configuration used by the client
type ClientConfig struct {
	ID            string
	ServerAddress string
	LoopAmount    int
	LoopPeriod    time.Duration
}

// BetMessage represents the bet information sent from the client.
type BetMessage struct {
	Nombre     string
	Apellido   string
	Documento  string
	Nacimiento string
	Numero     string
}

// ConfirmationMessage represents the confirmation returned by the server.
type ConfirmationMessage struct {
	Status    string
	Documento string
	Numero    string
	Message   string
}

// Client entity that encapsulates how the client works.
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

// CreateClientSocket initializes client socket.
// In case of failure, it logs the error and returns it.
func (c *Client) createClientSocket() error {
	conn, err := net.Dial("tcp", c.config.ServerAddress)
	if err != nil {
		log.Criticalf(
			"action: connect | result: fail | client_id: %v | error: %v",
			c.config.ID,
			err,
		)
		return err
	}
	c.conn = conn
	return nil
}

// serializeBet serializes a BetMessage into a string using '|' as a delimiter.
// This is my custom protocol for communication.
func serializeBet(b BetMessage) string {
	return b.Nombre + "|" + b.Apellido + "|" + b.Documento + "|" + b.Nacimiento + "|" + b.Numero
}

// parseConfirmation parses a confirmation message string into a ConfirmationMessage struct.
// The expected format is: status|documento|numero|message
func parseConfirmation(s string) (ConfirmationMessage, error) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, "|")
	if len(parts) < 3 {
		return ConfirmationMessage{}, fmt.Errorf("invalid confirmation message: %s", s)
	}
	var msg string
	if len(parts) > 3 {
		msg = parts[3]
	}
	return ConfirmationMessage{
		Status:    parts[0],
		Documento: parts[1],
		Numero:    parts[2],
		Message:   msg,
	}, nil
}

// StartClientBet sends the bet information to the server.
// This method implements the business logic for my lottery use-case.
// It reads bet data from environment variables, serializes the data using my custom protocol,
// sends it to the server (appending "\n" as delimiter to avoid short-read issues),
// and waits for a confirmation. It also handles SIGTERM for graceful termination.
func (c *Client) StartClientBet() {
	// Set up a signal channel to catch SIGTERM for graceful termination.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM)

	// Check if a SIGTERM signal has been received before proceeding.
	select {
	case <-sigChan:
		log.Infof("action: exit | result: success | client_id: %v | message: SIGTERM received", c.config.ID)
		return
	default:
		// No SIGTERM received; proceed.
	}

	// Read bet details from environment variables.
	nombre := os.Getenv("NOMBRE")
	apellido := os.Getenv("APELLIDO")
	documento := os.Getenv("DOCUMENTO")
	nacimiento := os.Getenv("NACIMIENTO")
	numero := os.Getenv("NUMERO")

	log.Debugf("action: read_bet_data | result: success | nombre: %v | apellido: %v | documento: %v | nacimiento: %v | numero: %v",
		nombre, apellido, documento, nacimiento, numero)

	// Create the bet message.
	bet := BetMessage{
		Nombre:     nombre,
		Apellido:   apellido,
		Documento:  documento,
		Nacimiento: nacimiento,
		Numero:     numero,
	}

	// Serialize the bet message using my custom protocol.
	serializedBet := serializeBet(bet)

	// Create the connection to the server.
	if err := c.createClientSocket(); err != nil {
		return
	}

	// Send the serialized bet message followed by a newline delimiter.
	_, err := c.conn.Write([]byte(serializedBet + "\n"))
	if err != nil {
		log.Errorf("action: send_bet | result: fail | error: %v", err)
		c.conn.Close()
		return
	}

	// Read the confirmation response from the server.
	response, err := bufio.NewReader(c.conn).ReadString('\n')
	c.conn.Close()
	if err != nil {
		log.Errorf("action: receive_confirmation | result: fail | error: %v", err)
		return
	}

	// Parse the confirmation response using my custom protocol.
	conf, err := parseConfirmation(response)
	if err != nil {
		log.Errorf("action: parse_confirmation | result: fail | error: %v", err)
		return
	}

	// Log the result of the bet submission.
	if conf.Status == "ok" {
		log.Infof("action: apuesta_enviada | result: success | dni: %v | numero: %v", conf.Documento, conf.Numero)
	} else {
		log.Errorf("action: apuesta_enviada | result: fail | message: %v", conf.Message)
	}
}
