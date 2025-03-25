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
        Server loop:
        - Accept a connection
        - Handle the batch of bets
        - If all bets are valid => success
        - If at least one fails => fail
        """
        while self._running:
            try:
                client_sock = self.__accept_new_connection()
            except OSError:
                break  # server was shutdown
            self.__handle_client_connection(client_sock)
        logging.info("action: server_shutdown | result: success | message: Server shutting down gracefully")

    def __handle_client_connection(self, client_sock):
        """
        Read a batch from client (multiple lines, ended by \n).
        If all lines are valid => store them with store_bets(...) => log success => respond success|count
        If any line is invalid => log fail => respond fail|0
        """
        try:
            # 1) Recibimos hasta 8KB y quitamos el \n final
            data = client_sock.recv(8192).decode('utf-8').rstrip('\n')
            if not data:
                logging.error("action: receive_batch | result: fail | reason: empty_data")
                client_sock.sendall("fail|0\n".encode('utf-8'))
                return

            # 2) Separamos el chunk por líneas
            lines = data.split('\n')

            # 3) Parseamos cada línea y creamos la lista de Bet
            bets_to_store = []
            for line in lines:
                fields = line.strip().split(',')
                if len(fields) != 5:
                    # Apuesta inválida => todo el batch falla
                    logging.info(f"action: apuesta_recibida | result: fail | cantidad: 0")
                    client_sock.sendall("fail|0\n".encode('utf-8'))
                    return

                # Asumimos: first_name, last_name, document, birthdate, number
                first_name, last_name, document, birthdate, number_str = fields

                # Por ejemplo, tomamos la "agency" desde la IP del cliente:
                addr = client_sock.getpeername()  # (ip, port)
                ip = addr[0]
                agency = ip.split('.')[-1]  # truco: último octeto como ID de agencia

                try:
                    bet = Bet(agency, first_name, last_name, document, birthdate, number_str)
                except Exception as e:
                    # Si la fecha o el número no parsean bien => batch fail
                    logging.info(f"action: apuesta_recibida | result: fail | cantidad: 0 | error: {str(e)}")
                    client_sock.sendall("fail|0\n".encode('utf-8'))
                    return

                bets_to_store.append(bet)

            # 4) Si todas las apuestas se pudieron parsear, las guardamos
            store_bets(bets_to_store)
            total = len(bets_to_store)
            logging.info(f"action: apuesta_recibida | result: success | cantidad: {total}")

            # 5) Responder success|count
            client_sock.sendall(f"success|{total}\n".encode('utf-8'))

        except Exception as e:
            logging.error(f"action: process_batch | result: fail | error: {str(e)}")
            client_sock.sendall("fail|0\n".encode('utf-8'))
        finally:
            client_sock.close()
            logging.info("action: close_connection | result: success | message: Client socket closed")

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """
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
