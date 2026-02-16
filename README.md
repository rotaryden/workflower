# Workflower (WIP)

Programmatic AI workflow boilerplate for boostraping and streamlining vibe-coded AI workflows.

As example, Suno automation server with Telegram integration and LLM-powered music generation added.

Note: Suno API is currently bogus :) it has no official API. WIP for workaround

Intended to be rolled out with special NGINX setup on a Linux instance.

Known Problems as of now:
- need to fix paths redirects cause Nginx etup assumes /tX prefix fro tool number X (port mapped 4000+X)

## Local Machine Prerequisites

- Go 1.21+
- Node.js 18+ (for running the suno-api server)
- OpenAI API key
- Suno account with cookie + 2Captcha API key (for music generation)
- Telegram bot token (optional, for notifications)
- Cloudflare tunnel (optional, for local webhook testing)

**Install cloudflared:**

```bash
# Linux
wget https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64
chmod +x cloudflared-linux-amd64
sudo mv cloudflared-linux-amd64 /usr/local/bin/cloudflared

# macOS
brew install cloudflared
```

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
rio ALL=(ALL) NOPASSWD: /bin/journalctl -u aiwf_*.service -f
```


## Suno API Setup

**Important:** Suno.ai does not have an official API. This project uses the third-party [suno-api](https://github.com/gcui-art/suno-api) server.

### Quick Setup

1. **Clone and install suno-api** (on the same VPS or locally):

```bash
git clone https://github.com/gcui-art/suno-api.git
cd suno-api
npm install
```

2. **Configure suno-api** - Create `.env` in the suno-api directory:

```env
SUNO_COOKIE=<your-cookie-from-suno.ai>
TWOCAPTCHA_KEY=<your-2captcha-api-key>
BROWSER=chromium
BROWSER_GHOST_CURSOR=false
BROWSER_LOCALE=en
BROWSER_HEADLESS=true
```

3. **Get your Suno cookie:**
   - Visit [suno.ai/create](https://suno.ai/create)
   - Open DevTools (F12) → Network tab → Refresh
   - Find request with `?__clerk_api_version`
   - Copy the entire `Cookie` header value

4. **Get 2Captcha API key:**
   - Sign up at [2captcha.com](https://2captcha.com) (or [rucaptcha.com](https://rucaptcha.com) for Russia/Belarus)
   - Top up your balance
   - Copy your API key

5. **Start the suno-api server:**

```bash
npm run dev
```

Test it: `curl http://localhost:3000/api/get_limit`

For detailed instructions, see [`lib/suno/README.md`](lib/suno/README.md).

## Configuration

### 1. Application Environment (`.env`)

```bash
cp .env_example .env
```

Edit `.env` with your settings

APP_NAME will be used ass binary name as well

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

Builds Linux AMD64 binary as `./build/your_app_name`

### Development Mode

Auto-reload on file changes (installs `air` if needed):

```bash
make dev
```

Access at `http://localhost:4000`

## Deploy

### Deploy to Remote Server

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

### 1. Local Testing with CLoudflare Tunnel

Expose local server via public HTTPS URL and auto-configure Telegram webhook:


```bash
./build/your_app_name -L
```

This will:
- Start Cloudflare tunnel (requires `cloudflared` installed)
- Get public HTTPS URL (e.g., `https://xyz.trycloudflare.com`)
- Override `BASE_URL` and `TELEGRAM_WEBHOOK_URL` automatically
- Register webhook with Telegram bot

**Note:** `BASE_URL` can include a path prefix (e.g., `https://example.com/api/workflower`). The application will automatically extract the path component and configure all routes, redirects, and URLs accordingly.


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
- `-setup` — [internal use] Run remote setup (used internally during deployment)

## Production Deployment Notes

### Running suno-api as a Service

For production, run suno-api as a systemd service. Create `/etc/systemd/system/suno-api.service`:

```ini
[Unit]
Description=Suno API Service
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/suno-api
Environment=NODE_ENV=production
ExecStart=/usr/bin/npm start
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable suno-api
sudo systemctl start suno-api
```

## Notes

- `OPENAI_API_KEY` is **required**
- **suno-api server must be running** before starting workflower (for music generation)
- Telegram features work without configuration (notifications disabled)
- Cloudflare tunnel requires `cloudflared` binary in PATH
- Deployment requires SSH access with key authentication
- Remote service auto-starts on server reboot (systemd)
- **Security:** Never expose the suno-api server to the public internet
