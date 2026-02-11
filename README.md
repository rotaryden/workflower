# Workflower

Suno AI workflow automation server with Telegram integration and LLM-powered music generation.

## Local Machine Prerequisites

- Go 1.21+
- OpenAI API key
- Suno API key (optional)
- Telegram bot token (optional, for notifications)
- Cloudflare tunnel (optional, for local webhook testing)

## VPS prerequisites

- Ubuntu 24.04+ server
- SSH access
- systemd
- Nginx
- Certbot

#### tune sudoers

Change

```bash
    sudo visudo -f /etc/sudoers.d/your_user
```

add the following line:

```bash
rio ALL=(ALL) NOPASSWD: /bin/systemctl ^(start|restart|enable|status) aiwf_.*\.service$
rio ALL=(ALL) NOPASSWD: /bin/systemctl daemon-reload         
rio ALL=(ALL) NOPASSWD: /bin/mv /tmp/aiwf_*.service /etc/systemd/system/aiwf_*.service
```


## Configuration

### 1. Application Environment (`.env`)

```bash
cp .env_example .env
```

Edit `.env` with your settings:

```bash
# Required
OPENAI_API_KEY=sk-your-openai-api-key-here

# Server
SERVER_PORT=4000
BASE_URL=http://yourdomain.com

# OpenAI
OPENAI_MODEL=gpt-4o

# Suno (optional)
SUNO_API_KEY=your-suno-api-key
SUNO_BASE_URL=https://studio-api.suno.ai

# Telegram (optional)
TELEGRAM_BOT_TOKEN=your-bot-token
TELEGRAM_CHAT_ID=your-chat-id
TELEGRAM_WEBHOOK_PATH=/telegram/webhook
TELEGRAM_WEBHOOK_SECRET=your-webhook-secret

# Features
ENABLE_PREMIUM_FEATURES=true
MAX_AUDIO_SIZE_MB=50
GIN_MODE=release
```

### 2. Deployment Environment (`.deploy.env`)

Required only for remote deployment:

```bash
cp .deploy.env_example .deploy.env
```

Edit `.deploy.env`:

```bash
REMOTE_HOST=user@your-server.com
SSH_PORT=22
SSH_KEY_PATH=/path/to/key  # Optional, uses system SSH config by default
```

## Build

### Local Build

```bash
make build
```

Builds Linux AMD64 binary as `./workflower`

### Development Mode

Auto-reload on file changes (installs `air` if needed):

```bash
make dev
```

## Run

### Standard Mode

```bash
./workflower
```

Access at `http://localhost:8080`

### Local Mode with Cloudflare Tunnel

Expose local server via public HTTPS URL and auto-configure Telegram webhook:

```bash
./workflower -L
```

This will:
- Start Cloudflare tunnel (requires `cloudflared` installed)
- Get public HTTPS URL (e.g., `https://xyz.trycloudflare.com`)
- Override `BASE_URL` and `TELEGRAM_WEBHOOK_URL` automatically
- Register webhook with Telegram bot

**Install cloudflared:**

```bash
# Linux
wget https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64
chmod +x cloudflared-linux-amd64
sudo mv cloudflared-linux-amd64 /usr/local/bin/cloudflared

# macOS
brew install cloudflared
```

## Deploy

### Deploy to Remote Server

```bash
./workflower -D
```

or

```bash
make deploy
```

This will:
1. Build Linux binary
2. SSH to remote server
3. Upload binary and `.env`
4. Create systemd service
5. Start/restart service

Remote service runs as: `/opt/aiworkflow/workflower/workflower`

### Check Remote Service

```bash
make remote-status
```

### View Remote Logs

```bash
make remote-logs
```

## Testing Telegram Integration

### 1. Local Testing with Tunnel

```bash
./workflower -L
```

Look for output:
```
🌐 Cloudflare tunnel active: https://xyz.trycloudflare.com
🔔 Telegram webhook URL: https://xyz.trycloudflare.com/telegram/webhook
✅ Telegram webhook registered
```

### 2. Test Webhook

Send message to your Telegram bot. Check terminal logs for webhook events.

### 3. Manual Webhook Check

```bash
curl https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getWebhookInfo
```

### 4. Delete Webhook (if needed)

```bash
curl https://api.telegram.org/bot<YOUR_BOT_TOKEN>/deleteWebhook
```

## Project Structure

```
workflower/
├── config/           # Configuration loader
├── handlers/         # HTTP handlers
├── lib/
│   ├── deploy/       # Deployment automation
│   ├── llm/          # OpenAI/OpenRouter clients
│   ├── suno/         # Suno API client
│   ├── telegram/     # Telegram bot/webhook
│   └── templating/   # Template helpers
├── storage/          # In-memory storage
├── templates/        # HTML templates & prompts
├── workflow/         # Workflow engine
└── main.go
```

## Useful Commands

```bash
# Build
make build

# Run with tunnel
./workflower -L

# Deploy
./workflower -D

# Clean build artifacts
make clean

# Format code
make fmt

# Download dependencies
make deps
```

## Flags

- `-D` — Deploy to remote server
- `-L` — Start with Cloudflare tunnel (local development)
- `-setup` — Run remote setup (used internally during deployment)

## Notes

- `OPENAI_API_KEY` is **required**
- Telegram features work without configuration (notifications disabled)
- Cloudflare tunnel requires `cloudflared` binary in PATH
- Deployment requires SSH access with key authentication
- Remote service auto-starts on server reboot (systemd)
