import socket
import ssl
from retrying import retry


@retry(stop_max_attempt_number=30, wait_fixed=2000)
def upgrade(domain, path):
    request = 'GET {0} HTTP/1.1\r\n' \
              'Host: {1}\r\n' \
              'Upgrade: websocket\r\n' \
              'Connection: Upgrade\r\n' \
              'Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n' \
              'Sec-WebSocket-Version: 13\r\n\r\n'.format(path, domain)
    context = ssl._create_unverified_context()
    with socket.create_connection((domain, 443), timeout=10) as plain:
        with context.wrap_socket(plain, server_hostname=domain) as secure:
            secure.sendall(request.encode())
            status = secure.recv(1024).decode().splitlines()[0]
    assert ' 101 ' in status, status
