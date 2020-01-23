from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.parse import urlparse
from urllib.parse import parse_qs
import logging
import os
from sys import argv
import jwt
from datetime import datetime, timezone
import html

debug_key = "debugKeyForTestOnlydNeverUseInProduction123456789012345678901234567890"

""" @Alex Foran: This is a part for backend.  It should redirect to url from get_redirect_url()
    it needs two things:
        shared key (coming from k8s environment)
        original resource url (comes with "orig_request" in request query)
"""

def create_token(signing_key=None, seconds_to_expire=60, **payload):
    """ Create signed token"""
    if signing_key is None:
        signing_key = os.environ.get("PROXY_SHARED_KEY")

    assert signing_key is not None, "Token signing key is not valid"
    payload["exp"] = int(datetime.now(tz=timezone.utc).timestamp()) + seconds_to_expire

    encoded_jwt = jwt.encode(payload, signing_key, algorithm="HS256")
    # print(encoded_jwt)
    # a = jwt.decode(encoded_jwt,signing_key)
    # print (a)
    return encoded_jwt


def get_signin_token(signing_key, orig_url, ret_token):
    """ Form redirect url adding signed token to the original url"""
    payload = {"resource": orig_url, "iss": ret_token}
    token = create_token(signing_key, 3600, **payload)
    return token.decode("utf-8")


""" end """


class HTTPServerHandler(BaseHTTPRequestHandler):
    def _set_response(self):
        self.send_response(200)
        self.send_header("Content-type", "text/html")
        self.end_headers()

    def do_GET(self):

        query_components = parse_qs(urlparse(self.path).query)

        ret_token = query_components.get("ret_token")
        orig_request = query_components.get("next")
        if (
            ret_token is not None
            and len(ret_token) > 0
            and orig_request is not None
            and len(orig_request) > 0
        ):
            token = get_signin_token(debug_key, orig_request[0], ret_token[0])
            self._set_response()
            self.wfile.write(
                "<!DOCTYPE html><html><head><title>Mock Signin</title></head><body>"
                f"<form action='//{html.escape(orig_request[0])}' method='GET'>"
                f"<input type='hidden' name='saturn_token' value='{html.escape(token)}'/>"
                "<input type='text' value='my-fake-username'/><br/><input type='password' value='password'/><br/>"
                "<button type='submit'>Sign In</button></form></body></html>"
                .encode("utf-8"))
        else:
            logging.info(
                "GET request,\nPath: %s\nHeaders:\n%s\n", str(self.path), str(self.headers)
            )
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

    server_address = ("0.0.0.0", port)
    httpd = HTTPServer(server_address, HTTPServerHandler)
    logging.info(f"Starting httpd on port {port}")
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        pass
    httpd.server_close()
    logging.info("Stopping httpd...")
