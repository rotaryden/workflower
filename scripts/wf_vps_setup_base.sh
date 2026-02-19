#!/bin/bash
set -euo pipefail

#==============================================================================
# Phase 1 VPS setup
# - nginx
# - certbot + SSL certificate
# - tools subdomain nginx config
# - PostgreSQL server + n8n database
#==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="${ENV_FILE:-$SCRIPT_DIR/.env}"

STATE_FILE="${STATE_FILE_BASE:-$HOME/wf_vps_setup_base.state}"
BACKUP_DIR="${BACKUP_DIR_BASE:-$HOME/wf_setup_base_backup_$(date +%Y%m%d_%H%M%S)}"

declare -A COMPLETED_STEPS
declare -a DOMAINS=()

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

trim_whitespace() {
    local value="$1"
    value="${value#"${value%%[![:space:]]*}"}"
    value="${value%"${value##*[![:space:]]}"}"
    echo "$value"
}

parse_subdomains() {
    local raw_subdomains="$1"
    local -a parsed_subdomains
    IFS=',' read -r -a parsed_subdomains <<< "$raw_subdomains"

    if [[ "${#parsed_subdomains[@]}" -lt 1 ]]; then
        log_error "SUBDOMAINS must contain at least one comma-separated value"
        exit 1
    fi

    DOMAINS=()
    local entry fqdn
    for entry in "${parsed_subdomains[@]}"; do
        entry="$(trim_whitespace "$entry")"
        if [[ -z "$entry" ]]; then
            log_error "SUBDOMAINS contains an empty value"
            exit 1
        fi

        if [[ "$entry" == *.* ]]; then
            fqdn="$entry"
        else
            fqdn="${entry}.${BASE_DOMAIN}"
        fi

        DOMAINS+=("$fqdn")
    done
}

load_env() {
    if [[ ! -f "$ENV_FILE" ]]; then
        log_error "Missing env file: $ENV_FILE"
        log_info "Copy $SCRIPT_DIR/.env.example to $SCRIPT_DIR/.env and fill values."
        exit 1
    fi

    set -a
    # shellcheck disable=SC1090
    source "$ENV_FILE"
    set +a

    local required_vars=(
        SUBDOMAINS
        TOOLS_SUBDOMAIN
        BASE_DOMAIN
        CERTBOT_EMAIL
        TOOLS_PORT
        TOOLS_NUMBER
        PG_VERSION
        PG_LOCAL_USER
        PG_LOCAL_PASSWORD
        PG_LOCAL_TOOLS_DB_NAME
        PG_LOCAL_TOOLS_DB_USER
        PG_LOCAL_TOOLS_DB_PASSWORD
    )

    for var_name in "${required_vars[@]}"; do
        if [[ -z "${!var_name:-}" ]]; then
            log_error "Required variable '$var_name' is empty in $ENV_FILE"
            exit 1
        fi
    done

    TOOLS_FULL_SUBDOMAIN="${TOOLS_SUBDOMAIN}.${BASE_DOMAIN}"

    parse_subdomains "$SUBDOMAINS"
}

load_state() {
    if [[ -f "$STATE_FILE" ]]; then
        while IFS='=' read -r key value; do
            if [[ -n "${key:-}" && ! "$key" =~ ^# ]]; then
                COMPLETED_STEPS["$key"]="$value"
            fi
        done < "$STATE_FILE"
        log_info "Loaded state from $STATE_FILE"
    fi
}

save_state() {
    mkdir -p "$(dirname "$STATE_FILE")"
    {
        echo "# Phase 1 state"
        echo "# Generated: $(date)"
        echo "BACKUP_DIR=$BACKUP_DIR"
        for step in "${!COMPLETED_STEPS[@]}"; do
            echo "$step=${COMPLETED_STEPS[$step]}"
        done
    } > "$STATE_FILE"
}

is_step_complete() {
    [[ "${COMPLETED_STEPS[$1]:-false}" == "true" ]]
}

mark_step_complete() {
    COMPLETED_STEPS["$1"]="true"
    save_state
    log_success "Step completed: $1"
}

ensure_backup_dir() {
    mkdir -p "$BACKUP_DIR"
}

backup_path() {
    local source="$1"
    local subdir="$2"
    local dest="$BACKUP_DIR/$subdir"
    ensure_backup_dir
    mkdir -p "$dest"

    if [[ -f "$source" ]]; then
        local name
        name="$(basename "$source")"
        sudo cp "$source" "$dest/${name}.bak" 2>/dev/null || cp "$source" "$dest/${name}.bak" 2>/dev/null || true
    elif [[ -d "$source" ]]; then
        local name
        name="$(basename "$source")"
        sudo cp -r "$source" "$dest/${name}.bakdir" 2>/dev/null || cp -r "$source" "$dest/${name}.bakdir" 2>/dev/null || true
    fi
}

check_command() {
    command -v "$1" >/dev/null 2>&1
}

preflight_checks() {
    if is_step_complete "preflight_checks"; then
        log_info "Skipping preflight_checks"
        return
    fi

    if [[ $EUID -eq 0 ]]; then
        log_error "Run as a normal user (not root)."
        exit 1
    fi
    sudo -v
    mark_step_complete "preflight_checks"
}

install_base_dependencies() {
    if is_step_complete "install_base_dependencies"; then
        log_info "Skipping install_base_dependencies"
        return
    fi

    log_info "Installing base dependencies (nginx/certbot/postgresql prerequisites)..."
    sudo apt-get update
    sudo apt-get install -y \
        ca-certificates curl gnupg lsb-release net-tools \
        nginx certbot python3-certbot-nginx postgresql-client

    mark_step_complete "install_base_dependencies"
}

install_postgresql_server() {
    if is_step_complete "install_postgresql_server"; then
        log_info "Skipping install_postgresql_server"
        return
    fi

    backup_path "/etc/postgresql/${PG_VERSION}/main/pg_hba.conf" "install_postgresql_server"

    log_info "Ensuring PostgreSQL repository exists..."
    sudo sh -c "echo \"deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main\" > /etc/apt/sources.list.d/pgdg.list"
    curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | sudo gpg --dearmor -o /etc/apt/trusted.gpg.d/postgresql.gpg

    sudo apt-get update
    sudo apt-get install -y "postgresql-${PG_VERSION}"
    sudo systemctl enable postgresql
    sudo systemctl start postgresql

    log_info "Setting password for PostgreSQL user '$PG_LOCAL_USER' from .env"
    sudo -u "$PG_LOCAL_USER" psql -c "ALTER USER $PG_LOCAL_USER WITH PASSWORD '$PG_LOCAL_PASSWORD';"

    sudo sed -i '/^local/s/peer/scram-sha-256/' "/etc/postgresql/${PG_VERSION}/main/pg_hba.conf"
    sudo systemctl restart postgresql

    mark_step_complete "install_postgresql_server"
}

setup_tools_database() {
    if is_step_complete "setup_tools_database"; then
        log_info "Skipping setup_tools_database"
        return
    fi

    log_info "Creating tools database/user in PostgreSQL..."
    PGPASSWORD="$PG_LOCAL_PASSWORD" psql -h 127.0.0.1 -U "$PG_LOCAL_USER" <<SQL
DO \$\$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_user WHERE usename = '${PG_LOCAL_TOOLS_DB_USER}') THEN
        CREATE USER ${PG_LOCAL_TOOLS_DB_USER} WITH PASSWORD '${PG_LOCAL_TOOLS_DB_PASSWORD}';
    END IF;
END
\$\$;

SELECT 'CREATE DATABASE ${PG_LOCAL_TOOLS_DB_NAME} OWNER ${PG_LOCAL_TOOLS_DB_USER}'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '${PG_LOCAL_TOOLS_DB_NAME}')\gexec

GRANT ALL PRIVILEGES ON DATABASE ${PG_LOCAL_TOOLS_DB_NAME} TO ${PG_LOCAL_TOOLS_DB_USER};
SQL

    mark_step_complete "setup_tools_database"
}

request_ssl_certificate() {
    if is_step_complete "request_ssl_certificate"; then
        log_info "Skipping request_ssl_certificate"
        return
    fi

    backup_path "/etc/letsencrypt/live/${BASE_DOMAIN}" "request_ssl_certificate"
    backup_path "/etc/letsencrypt/archive/${BASE_DOMAIN}" "request_ssl_certificate"
    backup_path "/etc/letsencrypt/renewal/${BASE_DOMAIN}.conf" "request_ssl_certificate"

    sudo systemctl stop nginx || true
    local certbot_domain_args=()
    local domain
    for domain in "${DOMAINS[@]}"; do
        certbot_domain_args+=("-d" "$domain")
    done

    sudo certbot certonly --standalone \
        --non-interactive --agree-tos \
        --email "$CERTBOT_EMAIL" \
        "${certbot_domain_args[@]}" \
        --cert-name "$BASE_DOMAIN"

    mark_step_complete "request_ssl_certificate"
}

nginx_ssl_config() {
    cat <<EOF
    ssl_certificate /etc/letsencrypt/live/${BASE_DOMAIN}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/${BASE_DOMAIN}/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
EOF
}

nginx_security_headers() {
    cat <<'EOF'
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
EOF
}

nginx_http_redirect() {
    local domain="$1"
    cat <<EOF
server {
    listen 80;
    listen [::]:80;
    server_name ${domain};
    return 301 https://\$server_name\$request_uri;
}
EOF
}

nginx_proxy_headers() {
    cat <<'EOF'
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
EOF
}

nginx_websocket_support() {
    cat <<'EOF'
        proxy_buffering off;
        proxy_request_buffering off;
EOF
}

nginx_standard_timeouts() {
    cat <<'EOF'
        proxy_connect_timeout 600s;
        proxy_send_timeout 600s;
        proxy_read_timeout 600s;
EOF
}

nginx_standard_limits() {
    cat <<'EOF'
        client_max_body_size 100M;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
        proxy_connect_timeout 60s;
EOF
}

create_nginx_default_reject_template() {
    cat <<EOF
server {
    listen 80 default_server;
    listen [::]:80 default_server;
    server_name _;
    return 444;
}

server {
    listen 443 ssl http2 default_server;
    listen [::]:443 ssl http2 default_server;
    server_name _;
    ssl_certificate /etc/letsencrypt/live/${BASE_DOMAIN}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/${BASE_DOMAIN}/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    return 444;
}
EOF
}

create_nginx_tools_template() {
    local upstreams=""
    local locations=""

    for ((i=0; i<TOOLS_NUMBER; i++)); do
        local port=$((TOOLS_PORT + i))
        upstreams+="upstream tools_backend_t${i} {
    server 127.0.0.1:${port};
    keepalive 32;
}

"
        locations+="    # Tool t${i} on port ${port}
    location /t${i}/ {
        proxy_pass http://tools_backend_t${i}/;

$(nginx_proxy_headers)
$(nginx_websocket_support)
$(nginx_standard_timeouts)
    }

"
    done

    cat <<EOF
${upstreams}
$(nginx_http_redirect "$TOOLS_FULL_SUBDOMAIN")

server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name ${TOOLS_FULL_SUBDOMAIN};

$(nginx_ssl_config)
$(nginx_security_headers)
$(nginx_standard_limits)

    # Root location - can serve a landing page or redirect
    location = / {
        return 200 '<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Tool Slots</title>
    <style>
        body { font-family: Arial, sans-serif; padding: 20px; background: #f5f5f5; }
        h1 { color: #333; text-align: center; margin-bottom: 30px; }
        .groups-container { display: grid; grid-template-columns: repeat(auto-fit, minmax(280px, 1fr)); gap: 20px; }
        .tool-group { background: white; border: 2px solid #ddd; border-radius: 8px; padding: 15px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .group-title { font-weight: bold; color: #555; margin: 0 0 12px 0; padding-bottom: 8px; border-bottom: 2px solid #e0e0e0; }
        .tools-list { list-style: none; padding: 0; margin: 0; }
        .tools-list li { margin: 4px 0; }
        .tools-list a { color: #333; text-decoration: none; padding: 12px 16px; display: block; text-align: center; border-radius: 4px; font-weight: 500; transition: all 0.2s; }
        .tools-list li:nth-child(odd) a { background-color: #f8f9fa; }
        .tools-list li:nth-child(even) a { background-color: #e9ecef; }
        .tools-list a:hover { background-color: #0066cc !important; color: white; transform: translateX(4px); }
    </style>
</head>
<body>
    <h1>Available Tools (${TOOLS_NUMBER})</h1>
    <div class="groups-container" id="groupsContainer"></div>
    <script>
        const container = document.getElementById("groupsContainer");
        const totalTools = ${TOOLS_NUMBER};
        const groupSize = 10;
        
        for (let groupStart = 0; groupStart < totalTools; groupStart += groupSize) {
            const groupEnd = Math.min(groupStart + groupSize - 1, ${TOOLS_NUMBER});
            const groupDiv = document.createElement("div");
            groupDiv.className = "tool-group";
            
            const groupTitle = document.createElement("h3");
            groupTitle.className = "group-title";
            groupTitle.textContent = "Tools /t" + groupStart + " - /t" + groupEnd;
            groupDiv.appendChild(groupTitle);
            
            const ul = document.createElement("ul");
            ul.className = "tools-list";
            
            for (let i = groupStart; i <= groupEnd && i <= ${TOOLS_NUMBER}; i++) {
                const li = document.createElement("li");
                const a = document.createElement("a");
                a.href = "/t" + i;
                a.textContent = "/t" + i;
                li.appendChild(a);
                ul.appendChild(li);
            }
            
            groupDiv.appendChild(ul);
            container.appendChild(groupDiv);
        }
    </script>
</body>
</html>';
        add_header Content-Type text/html;
    }

${locations}
}
EOF
}

configure_nginx_tools() {
    if is_step_complete "configure_nginx_tools"; then
        log_info "Skipping configure_nginx_tools"
        return
    fi

    backup_path "/etc/nginx/sites-available" "configure_nginx_tools"
    backup_path "/etc/nginx/sites-enabled" "configure_nginx_tools"
    backup_path "/etc/nginx/nginx.conf" "configure_nginx_tools"

    sudo rm -f /etc/nginx/sites-enabled/default

    create_nginx_default_reject_template | sudo tee /etc/nginx/sites-available/00-default-reject.conf >/dev/null
    create_nginx_tools_template | sudo tee /etc/nginx/sites-available/tools.conf >/dev/null

    sudo ln -sf /etc/nginx/sites-available/00-default-reject.conf /etc/nginx/sites-enabled/00-default-reject.conf
    sudo ln -sf /etc/nginx/sites-available/tools.conf /etc/nginx/sites-enabled/tools.conf

    sudo nginx -t
    sudo systemctl restart nginx

    mark_step_complete "configure_nginx_tools"
}

create_certbot_renewal() {
    if is_step_complete "create_certbot_renewal"; then
        log_info "Skipping create_certbot_renewal"
        return
    fi

    sudo tee /etc/systemd/system/certbot-renewal.service >/dev/null <<'EOF'
[Unit]
Description=Certbot SSL Certificate Renewal
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/bin/certbot renew --quiet --post-hook "systemctl reload nginx"
StandardOutput=journal
StandardError=journal
EOF

    sudo tee /etc/systemd/system/certbot-renewal.timer >/dev/null <<'EOF'
[Unit]
Description=Certbot SSL Certificate Renewal Timer
After=network.target

[Timer]
OnCalendar=*-*-* 00,12:00:00
RandomizedDelaySec=3600
Persistent=true

[Install]
WantedBy=timers.target
EOF

    sudo systemctl daemon-reload
    sudo systemctl enable certbot-renewal.timer
    sudo systemctl start certbot-renewal.timer

    mark_step_complete "create_certbot_renewal"
}

start_base_services() {
    if is_step_complete "start_base_services"; then
        log_info "Skipping start_base_services"
        return
    fi

    sudo systemctl enable postgresql
    sudo systemctl start postgresql
    sudo systemctl enable nginx
    sudo systemctl restart nginx
    sudo systemctl start certbot-renewal.timer

    mark_step_complete "start_base_services"
}

verify_base() {
    echo ""
    echo "==== Phase 1 verification ===="
    sudo systemctl is-active --quiet postgresql && echo "PostgreSQL: running" || echo "PostgreSQL: not running"
    sudo systemctl is-active --quiet nginx && echo "Nginx: running" || echo "Nginx: not running"
    sudo systemctl is-active --quiet certbot-renewal.timer && echo "Certbot timer: active" || echo "Certbot timer: inactive"
    echo "State file: $STATE_FILE"
    echo "Backup dir: $BACKUP_DIR"
    echo "=============================="
    echo ""
}

main() {
    load_env
    load_state
    preflight_checks
    install_base_dependencies
    install_postgresql_server
    setup_tools_database
    request_ssl_certificate
    configure_nginx_tools
    create_certbot_renewal
    start_base_services
    verify_base
    log_success "Phase 1 completed."
}

if [[ "${1:-}" == "verify" ]]; then
    load_env
    load_state
    verify_base
else
    main
fi
