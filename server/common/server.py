import socket
import logging
import threading

from common.utils import Bet, store_bets, load_bets, has_won
from common.protocol import read_message_with_length_prefix, parse_message

class Server:
    def __init__(self, port, listen_backlog, expected_agencies):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._running = True

        # Internal state
        self._notified_agencies = set()  # track which agencies have completed sending
        self.expected_agencies = expected_agencies # expected agencies (docker-compose clients)
        self._winners_by_agency = {}  # store winning documents per agency
        self._draw_done = False  # flag to avoid re-running the draw
        self._lock = threading.Lock()  # ensure thread-safe updates

        logging.info("action: server_start | result: success | message: Server started")

    def run(self):
        """
        Server loop:
        - Accept a connection.
        - Process each connection in a new thread.
        At shutdown, waits for all threads to finish.
        """
        threads = []
        while self._running:
            try:
                client_sock = self.__accept_new_connection()
            except OSError:
                break
            # Create a new thread to handle the connection
            t = threading.Thread(target=self.__handle_client_connection, args=(client_sock,))
            t.start()
            threads.append(t)
        
        # Wait for all threads to complete before shutdown
        for t in threads:
            t.join()
            
        logging.info("action: server_shutdown | result: success | message: Server shutting down gracefully")


    def __accept_new_connection(self):
        """
        Accept new connections (blocking until a client connects).
        Returns the connected socket.
        """
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c
    
    def __read_exactly(self, client_sock, n: int) -> bytes:
        """
        Reads exactly n bytes from the socket.
        Raises an exception if the connection is closed before n bytes are read.
        """
        data = bytearray()
        while len(data) < n:
            chunk = client_sock.recv(n - len(data))
            if not chunk:
                raise Exception("Connection closed before reading expected bytes")
            data.extend(chunk)
        return bytes(data)
    
    def read_message_with_length_prefix(self, client_sock, delimiter: bytes = b";") -> str:
        """
        Reads the message sent over the socket using a length prefix.
        First, it reads the header (until the delimiter) to obtain the expected message length,
        then it reads exactly that many bytes from the socket.
        
        Returns:
            The complete message as a UTF-8 string (with the final newline removed).
        """
        header_bytes = bytearray()
        # Read header until delimiter is found
        while True:
            byte = client_sock.recv(1)
            if not byte:
                break
            if byte == delimiter:
                break
            header_bytes.extend(byte)
        
        if not header_bytes:
            return ""
        
        try:
            expected_length = int(header_bytes.decode("utf-8"))
        except Exception as e:
            logging.error(f"Invalid header received: {header_bytes}")
            return ""
        
        # Read exactly the expected number of bytes
        try:
            message_bytes = self.__read_exactly(client_sock, expected_length)
        except Exception as e:
            logging.error(f"Error reading message body: {str(e)}")
            return ""
        
        # Decode and remove trailing newline (if any)
        return message_bytes.decode("utf-8").rstrip("\n")


    def __handle_client_connection(self, client_sock):
        """
        Handles a new connection from a client.
        Possible messages:
          1) "notify_finished|<agency_id>"
          2) "query_winners|<agency_id>"
          3) A batch of bets (header: "agency_ID|<id>", followed by lines "A,B,doc,2000-01-01,num")
        """
        try:
            data = read_message_with_length_prefix(client_sock)
            message_info = parse_message(data)

            # If something is wrong or empty
            if message_info.get('type') == 'error':
                reason = message_info.get('reason', 'unknown_error')
                logging.error(f"action: receive_message | result: fail | reason: {reason}")
                client_sock.sendall(b"fail|0\n")
                return

            msg_type = message_info['type']

            if msg_type == 'notify_finished':
                agency = message_info['agency']
                self._handle_notify_finished(agency)
                client_sock.sendall(b"ack_notify\n")
                return

            if msg_type == 'query_winners':
                agency = message_info['agency']
                self._handle_query_winners(client_sock, agency)
                return

            if msg_type == 'batch':
                agency = message_info['agency']
                logging.info(f"action: parse_agency | result: success | agency: {agency}")

                bets_to_store = []
                for line in message_info['lines']:
                    fields = line.strip().split(',')
                    if len(fields) != 5:
                        logging.info("action: apuesta_recibida | result: fail | reason: bad_format")
                        client_sock.sendall(b"fail|0\n")
                        return

                    first_name, last_name, document, birthdate, number_str = fields
                    try:
                        bet = Bet(agency, first_name, last_name, document, birthdate, number_str)
                    except Exception as e:
                        logging.info(f"action: apuesta_recibida | result: fail | error: {str(e)}")
                        client_sock.sendall(b"fail|0\n")
                        return

                    bets_to_store.append(bet)

                # Store bets in a thread-safe manner
                with self._lock:
                    store_bets(bets_to_store)

                total = len(bets_to_store)
                logging.info(f"action: apuesta_recibida | result: success | cantidad: {total}")
                client_sock.sendall(f"success|{total}\n".encode('utf-8'))

        except Exception as e:
            logging.error(f"action: process_batch | result: fail | error: {str(e)}")
            client_sock.sendall("fail|0\n".encode('utf-8'))
        finally:
            client_sock.close()
            logging.info("action: close_connection | result: success | message: Client socket closed")

    def _handle_notify_finished(self, agency):
        """
        Called when a client sends "notify_finished|<agency>".
        We add that agency to notified set. If the number of notified agencies equals
        the number of expected agencies, we run the draw exactly once.
        """
        agency_id = int(agency)
        with self._lock:
            self._notified_agencies.add(agency_id)
            # Run the draw only once when all expected agencies have notified
            if not self._draw_done and (len(self._notified_agencies) == self.expected_agencies):
                all_bets = load_bets()
                for bet in all_bets:
                    if has_won(bet):
                        # store winner in winners_by_agency
                        agency = int(bet.agency)
                        self._winners_by_agency.setdefault(agency, []).append(bet.document)
                self._draw_done = True
                logging.info("action: sorteo | result: success")

    def _handle_query_winners(self, client_sock, agency):
        """
        If the draw is done, respond with the winners for that agency.
        Otherwise respond in_progress-sorteo_no_listo.
        """
        with self._lock:
            if not self._draw_done:
                client_sock.sendall("in_progress-sorteo_no_listo\n".encode('utf-8'))
                return

            agency_id = int(agency)
            winners = self._winners_by_agency.get(agency_id, [])
            count = len(winners)

            # Send "ok|count\n" as header
            header = f"ok|{count}\n"
            client_sock.sendall(header.encode('utf-8'))

            # Then each winning document on its own line
            for doc in winners:
                client_sock.sendall(f"{doc}\n".encode('utf-8'))

            logging.info(
            f"action: consulta_ganadores | result: success | cant_ganadores: {count} | agency: client{agency}"
            )

    def shutdown(self):
        """
        Graceful shutdown: stop the server loop, wait for all active threads to finish processing, and close the server socket.
        """
        self._running = False
        try:
            self._server_socket.close()
            logging.info("action: close_server_socket | result: success | message: Server socket closed")
        except Exception as e:
            logging.error(f"action: close_server_socket | result: fail | error: {e}")
