#!/bin/bash
set -e

# Load deployment configuration
if [ -f ".env" ] && [ -f ".deploy.env" ]; then
    echo "📝 Loading environment variables..."
    set -a
    source .env
    source .deploy.env
    set +a
else
    echo "⚠️  .env or .deploy.env not found. Please create it from .env_example and .deploy.env_example"
    exit 1
fi

# Load application configuration for APP_NAME
if [ -f ".env" ]; then
    set -a
    source .env
    set +a
fi

# Validate required variables
if [ -z "${REMOTE_HOST}" ]; then
    echo "❌ REMOTE_HOST not set in .deploy.env"
    exit 1
fi

if [ -z "${BASE_REMOTE_PATH}" ]; then
    echo "❌ BASE_REMOTE_PATH not set in .env"
    exit 1
fi

if [ -z "${APP_NAME}" ]; then
    echo "❌ APP_NAME not set in .env"
    exit 1
fi

# Deployment script for Suno Workflow Server
# Usage: ./deploy.sh [user@host] [remote_path]

REMOTE_PATH="${BASE_REMOTE_PATH}/${APP_NAME}"

echo "🚀 Building for Linux..."
GOOS=linux GOARCH=amd64 go build -o "${APP_NAME}" .

echo "📦 Deploying to ${REMOTE_HOST}:${REMOTE_PATH}..."

# Copy binary
scp "${APP_NAME}" "${REMOTE_HOST}:${REMOTE_PATH}/"

# Copy service file if it exists
if [ -f "${APP_NAME}.service" ]; then
    echo "📝 Copying service file..."
    scp "${APP_NAME}.service" "${REMOTE_HOST}:/tmp/${APP_NAME}.service"
fi

# Copy .env_example if it exists
if [ -f ".env_example" ]; then
    scp ".env_example" "${REMOTE_HOST}:${REMOTE_PATH}/"
fi

# Copy .env file if it exists
if [ -f ".env" ]; then
    echo "📝 Copying .env file..."
    scp ".env" "${REMOTE_HOST}:${REMOTE_PATH}/"
fi

# Make binary executable
ssh "${REMOTE_HOST}" "chmod +x ${REMOTE_PATH}/${APP_NAME}"

# Remote setup with checks
echo "🔧 Running remote setup..."
ssh "${REMOTE_HOST}" "bash -s" << EOF
set -e

# Check if systemd service exists and is enabled
SERVICE_EXISTS=\$(systemctl list-unit-files | grep -c "^${APP_NAME}.service" || true)
SERVICE_ENABLED=\$(systemctl is-enabled ${APP_NAME} 2>/dev/null || echo "not-found")

if [ "\$SERVICE_EXISTS" -eq 0 ] || [ "\$SERVICE_ENABLED" = "not-found" ]; then
    echo "🔧 Setting up ${APP_NAME} service..."
    
    # Create installation directory with proper permissions
    sudo mkdir -p ${REMOTE_PATH}/uploads
    sudo chown -R www-data:www-data ${REMOTE_PATH}
    
    # Install systemd service if it exists
    if [ -f "/tmp/${APP_NAME}.service" ]; then
        sudo mv /tmp/${APP_NAME}.service /etc/systemd/system/${APP_NAME}.service
        sudo systemctl daemon-reload
        echo "✅ Systemd service installed"
        
        # Enable the service
        sudo systemctl enable ${APP_NAME}
        echo "✅ Service enabled"
    else
        echo "⚠️  No service file found, skipping systemd setup"
    fi
else
    echo "✅ Service already configured and enabled"
fi

# Restart or start the service
if systemctl is-active --quiet ${APP_NAME}; then
    echo "🔄 Restarting service..."
    sudo systemctl restart ${APP_NAME}
else
    echo "🚀 Starting service..."
    sudo systemctl start ${APP_NAME}
fi

# Show status
echo ""
echo "📊 Service status:"
sudo systemctl status ${APP_NAME} --no-pager -l || true
EOF

echo ""
echo "✅ Deployment complete!"
echo ""
echo "📋 Useful commands:"
echo "  View logs: ssh ${REMOTE_HOST} 'sudo journalctl -u ${APP_NAME} -f'"
echo "  Check status: ssh ${REMOTE_HOST} 'sudo systemctl status ${APP_NAME}'"
echo "  Edit .env: ssh ${REMOTE_HOST} 'sudo nano ${REMOTE_PATH}/.env && sudo systemctl restart ${APP_NAME}'"

# Clean up local binary
rm -f "${APP_NAME}"

