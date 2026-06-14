"""Simple HTTP server with correct UTF-8 Content-Type headers for API docs."""

import http.server
import sys

PORT = int(sys.argv[1]) if len(sys.argv) > 1 else 8090

MIME_TYPES = {
    ".yaml": "text/yaml; charset=utf-8",
    ".yml": "text/yaml; charset=utf-8",
    ".html": "text/html; charset=utf-8",
    ".json": "application/json; charset=utf-8",
    ".css": "text/css; charset=utf-8",
    ".js": "application/javascript; charset=utf-8",
}


class Handler(http.server.SimpleHTTPRequestHandler):
    def end_headers(self):
        for ext, content_type in MIME_TYPES.items():
            if self.path.endswith(ext):
                self.send_header("Content-Type", content_type)
                break
        super().end_headers()


print(f"Serving API docs at http://localhost:{PORT}/swagger-ui.html")
http.server.HTTPServer(("", PORT), Handler).serve_forever()
