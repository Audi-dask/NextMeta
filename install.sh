#!/usr/bin/env bash
# NextMeta one-click installer.
# Downloads the latest deployment files from the GitHub repo and starts services.
set -euo pipefail

REPO_RAW="https://raw.githubusercontent.com/Audi-dask/NextMeta/main"
INSTALL_DIR="${1:-nextmeta}"

echo "==> Installing NextMeta into ./${INSTALL_DIR}"
mkdir -p "${INSTALL_DIR}"
cd "${INSTALL_DIR}"

echo "==> Fetching latest deployment files from GitHub..."
curl -fsSL -o docker-compose.yaml "${REPO_RAW}/docker-compose.yaml"
curl -fsSL -o init.sql "${REPO_RAW}/init.sql"
curl -fsSL -o config.example.yaml "${REPO_RAW}/config.example.yaml"

if [ -f config.yaml ]; then
  echo "==> config.yaml already exists, keep it as is"
else
  cp config.example.yaml config.yaml
  echo "==> Created config.yaml from config.example.yaml"
fi

echo "==> Starting services..."
if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  docker compose up -d
else
  docker-compose up -d
fi

echo ""
echo "NextMeta is starting, open http://localhost:8080 when ready."
echo "Default admin: NextMeta / password123 (change it after first login)."
echo ""
echo "[NOTE] The license file (license.lic) cannot be filled automatically."
echo "       Please upload it manually via System Settings after the service is up."
