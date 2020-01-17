from http.server import BaseHTTPRequestHandler, HTTPServer
import logging
from sys import argv


class HTTPServerHandler(BaseHTTPRequestHandler):
    def _set_response(self):
        self.send_response(200)
        self.send_header("Content-type", "text/html")
        self.end_headers()

    def do_GET(self):
        logging.info("GET request,\nPath: %s\nHeaders:\n%s\n", str(self.path), str(self.headers))
        self._set_response()
        self.wfile.write(f"GET request for {self.path}".encode("utf-8"))

    def do_POST(self):
        content_length = int(self.headers["Content-Length"])  # <--- Gets the size of data
        post_data = self.rfile.read(content_length)  # <--- Gets the data itself
        logging.info(
            "POST request,\nPath: %s\nHeaders:\n%s\n\nBody:\n%s\n",
            str(self.path),
            str(self.headers),
            post_data.decode("utf-8"),
        )

        self._set_response()
        self.wfile.write(f"POST request for {self.path}".encode("utf-8"))


if __name__ == "__main__":

    logging.basicConfig(level=logging.INFO)

    port = 8888
    if len(argv) == 2:
        port = int(argv[1])

    server_address = ("", port)
    httpd = HTTPServer(server_address, HTTPServerHandler)
    logging.info(f"Starting httpd on port {port}")
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        pass
    httpd.server_close()
    logging.info("Stopping httpd...")
