#!/usr/bin/env bash
# Deploy backend Finance: tarik kode terbaru, build, jalankan/-ulang via PM2.
# Jalankan di server dari dalam folder repo: ./deploy.sh
set -euo pipefail
cd "$(dirname "$0")"

echo "==> git pull"
git pull --ff-only

echo "==> go build"
export PATH="$PATH:/usr/local/go/bin"
CGO_ENABLED=0 go build -trimpath -o finance-server ./cmd/server

# Muat env (port + koneksi Postgres) dari file di luar git: /opt/apps/finance.env
set -a; [ -f /opt/apps/finance.env ] && . /opt/apps/finance.env; set +a

echo "==> (re)start PM2: finance-be"
pm2 restart finance-be --update-env 2>/dev/null || pm2 start ./finance-server --name finance-be --update-env
pm2 save
echo "==> selesai. status:"
pm2 status finance-be
