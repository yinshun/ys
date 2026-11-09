# -*- coding: utf-8 -*-
"""serve-local.py — 本機靜態伺服器（給 ys.adb 測試用，沒有 hot reload）

    python tools/serve-local.py [port]

服務 `D:\\accelon3`，加上 `Cache-Control: no-store`，所以重建 ys.adb 後重新整理
就會拿到新檔（不會被瀏覽器快取）。不做任何監看／自動重建，改檔要自己 refresh。

預設 http://127.0.0.1:8899/accelon26.html
"""
import http.server
import os
import socketserver
import sys

ROOT = r'D:\accelon3'


class Handler(http.server.SimpleHTTPRequestHandler):
    def end_headers(self):
        self.send_header('Cache-Control', 'no-store, max-age=0')
        super().end_headers()

    def log_message(self, fmt, *args):
        sys.stderr.write('%s - %s\n' % (self.address_string(), fmt % args))


def main():
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8899
    os.chdir(ROOT)
    socketserver.TCPServer.allow_reuse_address = True
    with socketserver.ThreadingTCPServer(('127.0.0.1', port), Handler) as srv:
        print('serving %s at http://127.0.0.1:%d/accelon26.html' % (ROOT, port), flush=True)
        srv.serve_forever()


if __name__ == '__main__':
    main()
