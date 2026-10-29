#!/usr/bin/env python3
"""TEST ONLY real embedded-IdP device flow. No setup-key member stand-ins."""
from html.parser import HTMLParser
import http.cookiejar
import json
import os
from pathlib import Path
import re
import socket
import subprocess
import time
import urllib.parse
import urllib.request

ROOT = Path(__file__).resolve().parent
STATE = ROOT / '.state'
PORT = int(os.environ.get('FVPN_NB_API_PORT', '18080'))
URL = f'http://netbird.example.invalid:{PORT}'


class Form(HTMLParser):
    def __init__(self):
        super().__init__()
        self.action = None
        self.fields = {}
        self.links = []

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if tag == 'form' and self.action is None:
            self.action = attrs.get('action', '')
        if tag == 'input' and attrs.get('name'):
            self.fields[attrs['name']] = attrs.get('value', '')
        if tag == 'a' and attrs.get('href'):
            self.links.append(attrs['href'])


def authorize(url, user):
    # Resolve only the documentation hostname to our published loopback socket.
    # This changes no host resolver configuration and performs no external DNS.
    original = socket.getaddrinfo
    def resolve(host, port, *args, **kwargs):
        if host != 'netbird.example.invalid' or int(port) != PORT:
            raise RuntimeError('Unexpected authentication destination')
        return original('127.0.0.1', port, *args, **kwargs)
    socket.getaddrinfo = resolve
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}),
                urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
    try:
        response = opener.open(url, timeout=15)
        for _ in range(10):
            html = response.read().decode()
            form = Form()
            form.feed(html)
            if form.action is None:
                # Some Dex versions present a connector selection first.
                link = next((s for s in form.links if '/auth/local' in s), None)
                if link:
                    response = opener.open(urllib.parse.urljoin(response.url, link), timeout=15)
                    continue
                if any(word in html.lower() for word in ('success', 'logged in', 'device is connected', 'device authorized')):
                    return
                raise RuntimeError('Unrecognized embedded IdP screen')
            fields = form.fields
            for name in ('login', 'email', 'username'):
                if name in fields:
                    fields[name] = user['email']
            if 'password' in fields:
                fields['password'] = user['password']
            if 'approval' in fields:
                fields['approval'] = 'approve'
            target = urllib.parse.urljoin(response.url, form.action)
            response = opener.open(target, urllib.parse.urlencode(fields).encode(), timeout=15)
        raise RuntimeError('Embedded IdP flow did not finish')
    finally:
        socket.getaddrinfo = original


def login(label):
    user = json.loads((STATE / f'member-{label}.json').read_text())
    compose = ['docker', 'compose', '-f', str(ROOT / 'compose.yml')]
    command = compose + ['exec', '-T', f'member-{label}', 'netbird', 'up',
              '--management-url', URL, '--hostname', f'fvpn-member-{label}',
              '--disable-dns', '--disable-ipv6', '--no-browser']
    path = STATE / f'member-{label}-login.log'
    with path.open('w') as output:
        process = subprocess.Popen(command, stdout=output, stderr=subprocess.STDOUT)
        try:
            for _ in range(90):
                text = path.read_text()
                match = re.search(r'http://netbird\.example\.invalid:\d+/\S+', text)
                if match:
                    authorize(match.group(), user)
                    break
                if process.poll() is not None:
                    if process.returncode != 0:
                        raise RuntimeError('Client login failed; private diagnostic log retained')
                    return
                time.sleep(1)
            else:
                raise RuntimeError('Client did not start device flow')
            if process.wait(timeout=30) != 0:
                raise RuntimeError('Client did not confirm login')
        finally:
            if process.poll() is None:
                process.terminate()
                process.wait(timeout=10)
    path.unlink(missing_ok=True)
    print(f'TEST ONLY member-{label}: real user device-flow login confirmed (credentials withheld).')


if __name__ == '__main__':
    os.umask(0o077)
    try:
        for label in ('a', 'b'):
            login(label)
    except Exception:
        print('Device-flow login failed; secrets withheld. See README troubleshooting.')
        raise SystemExit(1)
