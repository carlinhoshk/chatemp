#!/usr/bin/env bash
set -euo pipefail

# Deploy do ChatTemp em uma VM Oracle (Ubuntu).
# Uso:
#   ./deploy/scripts/deploy.sh
# Antes, edite deploy/.env e informe o IP/domínio da VM.

cd "$(dirname "${BASH_SOURCE[0]}")/../.." # raiz do repositório

if [ ! -f deploy/.env ]; then
  echo "==> Copiando deploy/.env.example para deploy/.env"
  cp deploy/.env.example deploy/.env
  echo "    Edite deploy/.env e defina DOMAIN com o IP/domínio da sua VM."
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "==> Docker não encontrado. Instalando..."
  curl -fsSL https://get.docker.com | sh
  sudo usermod -aG docker "$USER"
  echo "    Reconfigure a sessão (logout/login) para usar o docker sem sudo."
fi

echo "==> Subindo containers..."
cd deploy
docker compose up -d --build

domain="$(grep -E '^DOMAIN=' .env | cut -d= -f2- || true)"
echo
echo "==> ChatTemp no ar!"
echo "    Acesse: http://${domain:-<IP da VM>}"
echo "    Lembre-se de liberar as portas 80/443 na VCN Security List da Oracle."
