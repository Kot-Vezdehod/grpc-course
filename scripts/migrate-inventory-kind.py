#!/usr/bin/env python3
"""Run existing Inventory Goose migrations through a local port-forward."""
import base64
import json
import os
from pathlib import Path
import subprocess
import urllib.parse

root = Path(__file__).resolve().parents[1]
secret = json.loads(subprocess.check_output(
    ['kubectl', 'get', 'secret', 'inventory-db', '-o', 'json']))
dsn = base64.b64decode(secret['data']['INVENTORY_DATABASE_URL']).decode()
url = urllib.parse.urlsplit(dsn)
port = int(os.environ.get('INVENTORY_LOCAL_DB_PORT', '15434'))
credentials = url.netloc.rsplit('@', 1)[0]
local_dsn = urllib.parse.urlunsplit(url._replace(netloc=f'{credentials}@127.0.0.1:{port}'))
env = dict(os.environ, GOOSE_DRIVER='postgres', GOOSE_DBSTRING=local_dsn)
subprocess.run(['goose', '-dir', str(root / 'migrations/inventory'), 'up'], env=env, check=True)
subprocess.run(['goose', '-dir', str(root / 'migrations/inventory'), 'status'], env=env, check=True)
