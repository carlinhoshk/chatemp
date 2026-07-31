# ChatTemp

Chat web de **conversas temporárias** com mídia que desaparece. Sem cadastro:
cada visitante recebe uma identidade em hash e entra em salas por código/link.

- **Salas expiram** após um tempo configurável (padrão 24h) — tudo é apagado (DB + arquivos).
- **Fotos/vídeos efêmeros** (🔥) somem alguns segundos depois de serem vistos — apagados no servidor, não só escondidos no cliente.
- **Identidade anônima**: o servidor só conhece o SHA-256 do id gerado no navegador (exibido como `f3a8…9c41`).

## Stack

| Camada | Tecnologia |
| --- | --- |
| Backend | Go 1.26, SQLite (`modernc.org/sqlite`, sem CGO), gorilla/websocket |
| Frontend | React 19, Vite, Tailwind CSS v4, TypeScript |
| Deploy | Docker Compose + nginx (reverse proxy, WS, uploads, TLS futuro) |

## Estrutura

```
backend/            API REST + WebSocket
  cmd/server/       ponto de entrada
  internal/         config, store (sqlite), ws hub, handlers, cleanup, storage
frontend/           app React (Vite)
deploy/             docker-compose, nginx, script de deploy, .env.example
Dockerfile          build multi-stage (frontend + backend → 1 imagem)
```

## Rodando em desenvolvimento

Terminal 1 — backend (API na porta 8080):

```bash
cd backend
go run ./cmd/server
```

Terminal 2 — frontend (Vite na porta 5173, com proxy para `/api` e `/ws`):

```bash
cd frontend
npm install
npm run dev
```

Abra http://localhost:5173. Crie uma sala e abra o link em outra aba/navegador
para conversar entre dois usuários.

### Configuração (variáveis de ambiente)

| Variável | Padrão | Descrição |
| --- | --- | --- |
| `PORT` | `8080` | Porta do backend |
| `DB_PATH` | `./data/chatemp.db` | Arquivo SQLite |
| `UPLOAD_DIR` | `./data/uploads` | Pasta de mídia |
| `STATIC_DIR` | vazio | Build do frontend (usado em produção) |
| `ROOM_TTL` | `24h` | Vida das salas |
| `EPHEMERAL_TTL` | `15s` | Tempo pós-visualização da mídia efêmera |
| `MEDIA_SWEEP` | `1m` | Sweep de segurança da mídia efêmera |
| `ROOM_PURGE` | `30m` | Purga física de salas expiradas |
| `MAX_UPLOAD_BYTES` | `52428800` | Upload máximo (50 MB) |

## Testes

```bash
cd backend && go test ./...
cd frontend && npm run build
```

O backend tem testes de integração cobrindo o fluxo completo: criação de sala,
chat por WebSocket, upload/serve de mídia e consumo da mídia efêmera.

## Deploy em uma VM Oracle Cloud

1. **Crie a instância** (Ubuntu) e na **VCN Security List** libere as portas
   `80` e `443` (Inbound TCP) — o passo que mais impede o acesso externo.

2. **Clone e suba** (instala Docker automaticamente se faltar):

   ```bash
   git clone <url-do-seu-repo> chatemp && cd chatemp
   cp deploy/.env.example deploy/.env
   # edite deploy/.env e defina DOMAIN=<IP público da VM ou domínio>
   ./deploy/scripts/deploy.sh
   ```

3. Acesse `http://<IP-da-VM>`.

O `deploy/nginx/chatemp.conf.template` usa o IP/domínio definido no `.env`.
O SQLite e os uploads ficam num volume Docker (`chatemp-data`), então os dados
sobrevivem a `docker compose down`/`up`.

### HTTPS (depois)

Com um domínio apontando para a VM:

```bash
sudo apt install -y certbot
sudo certbot certonly --standalone -d seu.dominio.com
```

Depois siga as instruções comentadas no template nginx para ativar o bloco
`listen 443 ssl`, ou use um container `certbot` no compose.

## Privacidade

- Códigos de sala aleatórios (8 chars) dificultam acesso por adivinhação.
- A identidade nunca é enviada em claro ao servidor — só o hash.
- Mídia efêmera é **deletada no servidor** após a visualização; salas expiradas
  têm DB e arquivos apagados de forma integral.
