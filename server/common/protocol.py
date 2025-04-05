# protocol.py
import logging
import socket

def read_exactly(client_sock: socket.socket, n: int) -> bytes:
    """
    Reads exactly n bytes from the client_sock.
    Raises an exception if the connection is closed before reading n bytes.
    """
    data = bytearray()
    while len(data) < n:
        chunk = client_sock.recv(n - len(data))
        if not chunk:
            raise Exception("Connection closed before reading the expected bytes")
        data.extend(chunk)
    return bytes(data)

def read_message_with_length_prefix(client_sock: socket.socket, delimiter: bytes = b";") -> str:
    """
    Reads a message from the client using a length prefix:
    1) Reads the header until it finds the delimiter (e.g. ";").
    2) Converts that header into an integer indicating how many bytes to read.
    3) Reads exactly those bytes from the socket.
    4) Returns the decoded string, stripping any trailing newline.
    If something fails or the data is invalid, returns an empty string.
    """
    header_bytes = bytearray()
    
    # Read until the delimiter is encountered
    while True:
        byte = client_sock.recv(1)
        if not byte:
            # No data: the client may have closed the connection
            break
        if byte == delimiter:
            # Found the delimiter
            break
        header_bytes.extend(byte)

    if not header_bytes:
        return ""

    try:
        expected_length = int(header_bytes.decode("utf-8"))
    except Exception as e:
        logging.error(f"Invalid header received: {header_bytes}")
        return ""

    # Read exactly the expected_length bytes
    try:
        message_bytes = read_exactly(client_sock, expected_length)
    except Exception as e:
        logging.error(f"Error reading message body: {str(e)}")
        return ""

    # Decode and remove any trailing newline
    return message_bytes.decode("utf-8").rstrip("\n")

def parse_message(data: str) -> dict:
    """
    Interprets the received data and categorizes it as one of:
      1) notify_finished|<agency>
      2) query_winners|<agency>
      3) batch: "agency_ID|<id>" plus subsequent lines with bets "X,Y,doc,YYYY-MM-DD,Z"
    
    Returns a dictionary with a 'type' key and any additional data needed.
    For example:
      { 'type': 'notify_finished', 'agency': '1' }
      { 'type': 'query_winners',  'agency': '2' }
      { 'type': 'batch', 'agency': '1', 'lines': [ ... ] }
    
    If it cannot parse properly, returns { 'type': 'error' } or similar.
    """
    if not data:
        return { 'type': 'error', 'reason': 'empty_data' }

    # 1) notify_finished|<agency>
    if data.startswith("notify_finished|"):
        parts = data.split('|', maxsplit=1)
        if len(parts) == 2:
            agency = parts[1].strip()
            return { 'type': 'notify_finished', 'agency': agency }
        else:
            return { 'type': 'error', 'reason': 'invalid_notify_finished' }

    # 2) query_winners|<agency>
    if data.startswith("query_winners|"):
        parts = data.split('|', maxsplit=1)
        if len(parts) == 2:
            agency = parts[1].strip()
            return { 'type': 'query_winners', 'agency': agency }
        else:
            return { 'type': 'error', 'reason': 'invalid_query_winners' }

    # 3) Otherwise, assume it's a batch starting with "agency_ID|<id>"
    lines = data.split('\n')
    if not lines or not lines[0].startswith("agency_ID|"):
        return { 'type': 'error', 'reason': 'missing_agency_id' }

    # The first line is "agency_ID|<id>"
    first_line = lines[0]
    _, agency_id_str = first_line.split('|', maxsplit=1)
    agency_id_str = agency_id_str.strip()

    # The rest of the lines are bets
    bet_lines = lines[1:]
    return {
        'type': 'batch',
        'agency': agency_id_str,
        'lines': bet_lines
    }
