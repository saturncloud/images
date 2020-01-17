from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.parse import urlparse
from urllib.parse import parse_qs
import logging
from sys import argv
import jwt
from datetime import datetime, timezone


debug_key = "debugKeyForTestOnlydNeverUseInProduction123456789012345678901234567890"


def create_token(signing_key= None,minutes=60, **payload):
    assert signing_key is not None, "Token signing key is not valid"
    payload["exp"] = int(datetime.now(tz=timezone.utc).timestamp()) + 60*minutes
    encoded_jwt = jwt.encode(payload, signing_key, algorithm='HS256')
    return encoded_jwt


class HTTPServerHandler(BaseHTTPRequestHandler):
    def _set_response(self):
        self.send_response(200)
        self.send_header("Content-type", "text/html")
        self.end_headers()

    def do_GET(self):

        query_components = parse_qs(urlparse(self.path).query)

        orig_request = query_components.get("orig_request")
        if orig_request is not None and len(orig_request)>0:
            payload = {"resource": orig_request[0]}
            token = create_token(
                debug_key,
                60,
                **payload
            )

            new_url = "http://"+orig_request[0]+"?saturn_token="+token.decode("utf-8")
            self.send_response(301)
            self.send_header('Location', new_url)
            self.end_headers()
        else:
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
