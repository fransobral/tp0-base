import socket
import logging
from common.utils import Bet, store_bets

class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._running = True
        logging.info("action: server_start | result: success | message: Server started")

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communication
        finishes, server starts to accept new connections again
        """
        while self._running:
            try:
                client_sock = self.__accept_new_connection()
            except OSError:
                break  # Server was shutdown
            self.__handle_client_connection(client_sock)
        logging.info("action: server_shutdown | result: success | message: Server shutting down gracefully")
    
    def __handle_client_connection(self, client_sock):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            # TODO: Modify the receive to avoid short-reads
            msg = client_sock.recv(1024).rstrip().decode('utf-8')
            addr = client_sock.getpeername()
            logging.info(f'action: receive_message | result: success | ip: {addr[0]} | msg: "{msg}"')

            # Parse message: nombre|apellido|documento|nacimiento|numero
            fields = msg.split('|')
            if len(fields) != 5:
                raise ValueError("Invalid bet format")

            nombre, apellido, documento, nacimiento, numero = fields
            agency = addr[0].split('.')[-1]  # estimate agency from last octate IP (placeholder)

            bet = Bet(agency, nombre, apellido, documento, nacimiento, numero)
            store_bets([bet])

            logging.info(f'action: apuesta_almacenada | result: success | dni: {documento} | numero: {numero}')

            response = f"ok|{documento}|{numero}\n"
            client_sock.sendall(response.encode('utf-8'))

        except Exception as e:
            logging.error(f"action: process_bet | result: fail | error: {str(e)}")
            try:
                client_sock.sendall(f"fail|||{str(e)}\n".encode('utf-8'))
            except:
                pass
        finally:
            client_sock.close()
            logging.info("action: close_connection | result: success | message: Client socket closed")

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c

    def shutdown(self):
        self._running = False
        try:
            self._server_socket.close()
            logging.info("action: close_server_socket | result: success | message: Server socket closed")
        except Exception as e:
            logging.error(f"action: close_server_socket | result: fail | error: {e}")