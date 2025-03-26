import socket
import logging
from common.utils import Bet, store_bets, load_bets, has_won
import threading

class Server:
    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._running = True

        # Internal state
        self._bets = []  # stores all received bets
        self._notified_agencies = set()  # track which agencies have completed sending
        self._seen_agencies = set() #track wich agencies have been seen
        self._winners_by_agency = {}  # store winning documents per agency
        self._draw_done = False  # flag to avoid re-running the draw
        self._lock = threading.Lock()  # ensure thread-safe updates

        logging.info("action: server_start | result: success | message: Server started")

    def run(self):
        """
        Server loop:
        - Accept a connection
        - Handle the incoming data:
          * notify_finished|<agency_id>
          * query_winners|<agency_id>
          * Or a batch of bets
        """
        while self._running:
            try:
                client_sock = self.__accept_new_connection()
            except OSError:
                break
            self.__handle_client_connection(client_sock)
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

    def __handle_client_connection(self, client_sock):
        """
        Handles a new connection from a client.
        Possible messages:
          1) "notify_finished|<agency_id>"
          2) "query_winners|<agency_id>"
          3) A batch of bets (header: "agency_ID|<id>", followed by lines "A,B,doc,2000-01-01,num")
        """
        try:
            data = client_sock.recv(8192).decode('utf-8').rstrip('\n')
            if not data:
                logging.error("action: receive_batch | result: fail | reason: empty_data")
                client_sock.sendall("fail|0\n".encode('utf-8'))
                return

            # 1) Handle "notify_finished|<agency>"
            if data.startswith("notify_finished|"):
                agency = data.split('|')[1].strip()
                self._handle_notify_finished(agency)
                client_sock.sendall("ack_notify\n".encode('utf-8'))
                return

            # 2) Handle "query_winners|<agency>"
            if data.startswith("query_winners|"):
                agency = data.split('|')[1].strip()
                self._handle_query_winners(client_sock, agency)
                return

            # 3) Otherwise, assume it's a batch starting with "agency_ID|<id>"
            lines = data.split('\n')
            if not lines[0].startswith("agency_ID|"):
                logging.error("action: parse_agency | result: fail | reason: missing_agency_id")
                client_sock.sendall("fail|0\n".encode('utf-8'))
                return

            # Extract agency ID from the first line
            agency = lines[0].split('|')[1].strip()
            logging.info(f"action: parse_agency | result: success | agency: {agency}")

            # Track that we have "seen" this agency (so we know how many total agencies are in play)
            with self._lock:
                self._seen_agencies.add(int(agency))

            # Parse each subsequent line as a single bet
            bets_to_store = []
            for line in lines[1:]:
                fields = line.strip().split(',')
                if len(fields) != 5:
                    logging.info("action: apuesta_recibida | result: fail | cantidad: 0 | reason: bad_format")
                    client_sock.sendall("fail|0\n".encode('utf-8'))
                    return

                # Parse individual bet fields
                first_name, last_name, document, birthdate, number_str = fields
                try:
                    # Attempt to construct a Bet object
                    bet = Bet(agency, first_name, last_name, document, birthdate, number_str)
                except Exception as e:
                    logging.info(f"action: apuesta_recibida | result: fail | error: {str(e)}")
                    client_sock.sendall("fail|0\n".encode('utf-8'))
                    return

                bets_to_store.append(bet)

            # Store the bets thread-safely
            with self._lock:
                store_bets(bets_to_store)
                self._bets.extend(bets_to_store)

            # Report successful receipt of bets
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
        We add that agency to notified set. If all seen agencies have notified,
        we run the draw exactly once.
        """
        agency_id = int(agency)
        with self._lock:
            self._notified_agencies.add(agency_id)

            # If all "seen" agencies have now notified, do the draw (only once)
            if not self._draw_done and (self._notified_agencies == self._seen_agencies):
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
        Otherwise respond fail|sorteo_no_listo
        """
        with self._lock:
            if not self._draw_done:
                client_sock.sendall("fail-sorteo_no_listo\n".encode('utf-8'))
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
            f"action: consulta_ganadores | result: success | cant_ganadores: {count}| agency: client{agency}"
            )

    def shutdown(self):
        """
        Graceful shutdown: stop the server loop and close the socket.
        """
        self._running = False
        try:
            self._server_socket.close()
            logging.info("action: close_server_socket | result: success | message: Server socket closed")
        except Exception as e:
            logging.error(f"action: close_server_socket | result: fail | error: {e}")
