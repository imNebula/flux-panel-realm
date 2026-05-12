#!/usr/bin/env bash
set -euo pipefail

REALM_VERSION="${REALM_VERSION:-v2.9.2-2}"
AGENT_VERSION="${AGENT_VERSION:-latest}"
REPO_OWNER="${REPO_OWNER:-imNebula}"
REPO_NAME="${REPO_NAME:-flux-panel-realm}"
SERVER_ADDR=""
SECRET=""
AGENT_NAME="flux-realm-agent"
AGENT_PROCESS_NAME="flux-realm-agent"
REALM_PROCESS_NAME="flux-realm"
SERVICE_NAME="flux-realm-agent"
INSTANCE="default"
MODE="single"
INSTALL_DIR="/opt/flux-realm-agent"
CONFIG_DIR=""
LOG_DIR="/var/log/flux-realm-agent"
DATA_DIR="/var/lib/flux-realm-agent"
NO_SYSTEM_SERVICE=0
FOREGROUND=0
CHINA_MIRROR=0
ACTION="install"
AGENT_URL="${AGENT_URL:-}"

usage() {
  cat <<EOF
Usage: $0 [install|update|uninstall|restart|status] [options]

Options:
  --server-addr HOST:PORT
  --secret SECRET
  --agent-name NAME
  --agent-process-name NAME
  --realm-process-name NAME
  --service-name NAME
  --install-dir DIR
  --config-dir DIR
  --log-dir DIR
  --instance NAME
  --mode single|per-tunnel|per-forward
  --no-system-service
  --foreground
  --china-mirror
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    install|update|uninstall|restart|status) ACTION="$1"; shift ;;
    -a|--server-addr) SERVER_ADDR="$2"; shift 2 ;;
    -s|--secret) SECRET="$2"; shift 2 ;;
    --agent-name) AGENT_NAME="$2"; shift 2 ;;
    --agent-process-name) AGENT_PROCESS_NAME="$2"; shift 2 ;;
    --realm-process-name) REALM_PROCESS_NAME="$2"; shift 2 ;;
    --service-name) SERVICE_NAME="$2"; shift 2 ;;
    --install-dir) INSTALL_DIR="$2"; shift 2 ;;
    --config-dir) CONFIG_DIR="$2"; shift 2 ;;
    --log-dir) LOG_DIR="$2"; shift 2 ;;
    --instance) INSTANCE="$2"; shift 2 ;;
    --mode) MODE="$2"; shift 2 ;;
    --no-system-service) NO_SYSTEM_SERVICE=1; shift ;;
    --foreground) FOREGROUND=1; shift ;;
    --china-mirror) CHINA_MIRROR=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown argument: $1" >&2; usage; exit 1 ;;
  esac
done

CONFIG_DIR="${CONFIG_DIR:-/etc/flux-realm/instances/${INSTANCE}}"
PID_FILE="${DATA_DIR}/${INSTANCE}.pid"
ENV_FILE="${CONFIG_DIR}/agent.env"

need_root() {
  if [[ "$(id -u)" != "0" ]]; then
    echo "This action requires root." >&2
    exit 1
  fi
}

cmd_exists() { command -v "$1" >/dev/null 2>&1; }

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "x86_64" ;;
    aarch64|arm64) echo "aarch64" ;;
    armv7l|armv7*) echo "armv7" ;;
    armv6l|arm*) echo "arm" ;;
    i386|i686) echo "i686" ;;
    *) echo "unsupported"; return 1 ;;
  esac
}

detect_libc() {
  if ldd --version 2>&1 | grep -qi musl; then echo "musl"; else echo "gnu"; fi
}

realm_asset() {
  local arch libc
  arch="$(detect_arch)"
  libc="$(detect_libc)"
  case "${arch}-${libc}" in
    x86_64-musl) echo "realm-x86_64-unknown-linux-musl.tar.gz" ;;
    x86_64-gnu) echo "realm-x86_64-unknown-linux-gnu.tar.gz" ;;
    aarch64-musl) echo "realm-aarch64-unknown-linux-musl.tar.gz" ;;
    aarch64-gnu) echo "realm-aarch64-unknown-linux-gnu.tar.gz" ;;
    armv7-musl) echo "realm-armv7-unknown-linux-musleabihf.tar.gz" ;;
    armv7-gnu) echo "realm-armv7-unknown-linux-gnueabihf.tar.gz" ;;
    arm-musl) echo "realm-arm-unknown-linux-musleabihf.tar.gz" ;;
    arm-gnu) echo "realm-arm-unknown-linux-gnueabihf.tar.gz" ;;
    i686-gnu) echo "realm-i686-unknown-linux-gnu.tar.gz" ;;
    *) echo "unsupported"; return 1 ;;
  esac
}

agent_asset() {
  case "$(detect_arch)" in
    x86_64) echo "flux-realm-agent-amd64" ;;
    aarch64) echo "flux-realm-agent-arm64" ;;
    armv7) echo "flux-realm-agent-armv7" ;;
    *) echo "unsupported"; return 1 ;;
  esac
}

download() {
  local url="$1" out="$2"
  if [[ "$CHINA_MIRROR" == "1" ]]; then
    url="https://ghfast.top/${url}"
  fi
  if cmd_exists curl; then
    curl -fsSL "$url" -o "$out"
  elif cmd_exists wget; then
    wget -qO "$out" "$url"
  else
    echo "curl or wget is required." >&2
    exit 1
  fi
}

install_realm() {
  local asset url tmp
  asset="$(realm_asset)"
  tmp="$(mktemp -d)"
  url="https://github.com/zhboner/realm/releases/download/${REALM_VERSION}/${asset}"
  echo "Downloading Realm ${REALM_VERSION} ${asset}"
  download "$url" "${tmp}/realm.tar.gz"
  tar -xzf "${tmp}/realm.tar.gz" -C "$tmp"
  local bin
  bin="$(find "$tmp" -type f -name realm -perm -111 | head -1)"
  [[ -n "$bin" ]] || bin="$(find "$tmp" -type f -name realm | head -1)"
  if [[ -z "$bin" ]]; then
    echo "Realm binary not found in archive." >&2
    exit 1
  fi
  install -m 0755 "$bin" "${INSTALL_DIR}/realm"
  rm -rf "$tmp"
}

install_agent() {
  local asset release_path
  if [[ -n "$AGENT_URL" ]]; then
    download "$AGENT_URL" "${INSTALL_DIR}/flux-realm-agent"
    chmod 0755 "${INSTALL_DIR}/flux-realm-agent"
    return
  fi
  if [[ -x "./flux-realm-agent" ]]; then
    install -m 0755 "./flux-realm-agent" "${INSTALL_DIR}/flux-realm-agent"
    return
  fi
  if [[ -x "./realm-agent/flux-realm-agent" ]]; then
    install -m 0755 "./realm-agent/flux-realm-agent" "${INSTALL_DIR}/flux-realm-agent"
    return
  fi
  asset="$(agent_asset)"
  if [[ "$AGENT_VERSION" == "latest" ]]; then
    release_path="latest/download"
  else
    release_path="download/${AGENT_VERSION}"
  fi
  download "https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/${release_path}/${asset}" "${INSTALL_DIR}/flux-realm-agent"
  chmod 0755 "${INSTALL_DIR}/flux-realm-agent"
}

write_env() {
  umask 077
  cat > "$ENV_FILE" <<EOF
SERVER_ADDR='${SERVER_ADDR}'
SECRET='${SECRET}'
AGENT_NAME='${AGENT_NAME}'
AGENT_PROCESS_NAME='${AGENT_PROCESS_NAME}'
REALM_PROCESS_NAME='${REALM_PROCESS_NAME}'
SERVICE_NAME='${SERVICE_NAME}'
INSTANCE='${INSTANCE}'
MODE='${MODE}'
INSTALL_DIR='${INSTALL_DIR}'
CONFIG_DIR='${CONFIG_DIR}'
LOG_DIR='${LOG_DIR}'
DATA_DIR='${DATA_DIR}'
PID_FILE='${PID_FILE}'
REALM_BINARY='${INSTALL_DIR}/realm'
EOF
}

agent_cmd() {
  printf "%q " "${INSTALL_DIR}/flux-realm-agent" \
    --agent-name "$AGENT_NAME" \
    --agent-process-name "$AGENT_PROCESS_NAME" \
    --realm-process-name "$REALM_PROCESS_NAME" \
    --service-name "$SERVICE_NAME" \
    --install-dir "$INSTALL_DIR" \
    --config-dir "$CONFIG_DIR" \
    --log-dir "$LOG_DIR" \
    --data-dir "$DATA_DIR" \
    --pid-file "$PID_FILE" \
    --instance "$INSTANCE" \
    --mode "$MODE" \
    --realm-binary "${INSTALL_DIR}/realm"
}

export_runtime_env() {
  export SERVER_ADDR SECRET AGENT_NAME AGENT_PROCESS_NAME REALM_PROCESS_NAME SERVICE_NAME
  export INSTANCE MODE INSTALL_DIR CONFIG_DIR LOG_DIR DATA_DIR PID_FILE
  export REALM_BINARY="${INSTALL_DIR}/realm"
}

detect_init() {
  if cmd_exists systemctl && systemctl list-unit-files >/dev/null 2>&1; then echo systemd; return; fi
  if cmd_exists rc-service || [[ -d /run/openrc ]]; then echo openrc; return; fi
  if cmd_exists supervisord || cmd_exists supervisorctl; then echo supervisord; return; fi
  echo none
}

install_service() {
  local init
  init="$(detect_init)"
  case "$init" in
    systemd)
      cat > "/etc/systemd/system/${SERVICE_NAME}.service" <<EOF
[Unit]
Description=Flux Realm Agent (${INSTANCE})
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=${ENV_FILE}
ExecStart=${INSTALL_DIR}/flux-realm-agent --agent-name \${AGENT_NAME} --agent-process-name \${AGENT_PROCESS_NAME} --realm-process-name \${REALM_PROCESS_NAME} --service-name \${SERVICE_NAME} --install-dir \${INSTALL_DIR} --config-dir \${CONFIG_DIR} --log-dir \${LOG_DIR} --data-dir \${DATA_DIR} --pid-file \${PID_FILE} --instance \${INSTANCE} --mode \${MODE} --realm-binary ${INSTALL_DIR}/realm
Restart=always
RestartSec=5
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
EOF
      systemctl daemon-reload
      systemctl enable --now "${SERVICE_NAME}"
      ;;
    openrc)
      cat > "/etc/init.d/${SERVICE_NAME}" <<EOF
#!/sbin/openrc-run
name="${SERVICE_NAME}"
command="${INSTALL_DIR}/flux-realm-agent"
export SERVER_ADDR='${SERVER_ADDR}'
export SECRET='${SECRET}'
export REALM_BINARY='${INSTALL_DIR}/realm'
command_args="--agent-name ${AGENT_NAME} --agent-process-name ${AGENT_PROCESS_NAME} --realm-process-name ${REALM_PROCESS_NAME} --service-name ${SERVICE_NAME} --install-dir ${INSTALL_DIR} --config-dir ${CONFIG_DIR} --log-dir ${LOG_DIR} --data-dir ${DATA_DIR} --pid-file ${PID_FILE} --instance ${INSTANCE} --mode ${MODE} --realm-binary ${INSTALL_DIR}/realm"
command_background=true
pidfile="${DATA_DIR}/${SERVICE_NAME}.openrc.pid"
output_log="${LOG_DIR}/agent.log"
error_log="${LOG_DIR}/agent.err"
depend() { need net; }
EOF
      chmod +x "/etc/init.d/${SERVICE_NAME}"
      rc-update add "${SERVICE_NAME}" default || true
      rc-service "${SERVICE_NAME}" restart
      ;;
    supervisord)
      mkdir -p /etc/supervisor/conf.d
      cat > "/etc/supervisor/conf.d/${SERVICE_NAME}.conf" <<EOF
[program:${SERVICE_NAME}]
command=$(agent_cmd)
environment=SERVER_ADDR="${SERVER_ADDR}",SECRET="${SECRET}",REALM_BINARY="${INSTALL_DIR}/realm"
autostart=true
autorestart=true
stdout_logfile=${LOG_DIR}/agent.log
stderr_logfile=${LOG_DIR}/agent.err
EOF
      supervisorctl reread || true
      supervisorctl update || true
      supervisorctl restart "${SERVICE_NAME}" || true
      ;;
    none)
      start_noinit
      ;;
  esac
}

start_noinit() {
  mkdir -p "$LOG_DIR" "$DATA_DIR"
  if [[ -f "${DATA_DIR}/${SERVICE_NAME}.agent.pid" ]] && kill -0 "$(cat "${DATA_DIR}/${SERVICE_NAME}.agent.pid")" 2>/dev/null; then
    return
  fi
  nohup $(agent_cmd) >> "${LOG_DIR}/agent.log" 2>> "${LOG_DIR}/agent.err" &
  echo $! > "${DATA_DIR}/${SERVICE_NAME}.agent.pid"
}

stop_instance() {
  local init
  init="$(detect_init)"
  case "$init" in
    systemd) systemctl stop "${SERVICE_NAME}" 2>/dev/null || true; systemctl disable "${SERVICE_NAME}" 2>/dev/null || true ;;
    openrc) rc-service "${SERVICE_NAME}" stop 2>/dev/null || true; rc-update del "${SERVICE_NAME}" default 2>/dev/null || true ;;
    supervisord) supervisorctl stop "${SERVICE_NAME}" 2>/dev/null || true ;;
  esac
  if [[ -f "${DATA_DIR}/${SERVICE_NAME}.agent.pid" ]]; then
    local pid
    pid="$(cat "${DATA_DIR}/${SERVICE_NAME}.agent.pid")"
    kill "$pid" 2>/dev/null || true
    rm -f "${DATA_DIR}/${SERVICE_NAME}.agent.pid"
  fi
  if [[ -f "$PID_FILE" ]]; then
    local rpid
    rpid="$(cat "$PID_FILE")"
    kill "$rpid" 2>/dev/null || true
    rm -f "$PID_FILE"
  fi
}

do_install() {
  need_root
  [[ "$MODE" =~ ^(single|per-tunnel|per-forward)$ ]] || { echo "invalid --mode" >&2; exit 1; }
  [[ -n "$SERVER_ADDR" && -n "$SECRET" ]] || { echo "--server-addr and --secret are required" >&2; exit 1; }
  mkdir -p "$INSTALL_DIR" "$CONFIG_DIR" "$LOG_DIR" "$DATA_DIR"
  install_realm
  install_agent
  write_env
  export_runtime_env
  if [[ "$FOREGROUND" == "1" ]]; then
    exec $(agent_cmd)
  fi
  if [[ "$NO_SYSTEM_SERVICE" == "1" ]]; then
    start_noinit
  else
    install_service
  fi
  "${INSTALL_DIR}/realm" --version || true
  echo "Installed ${SERVICE_NAME} instance=${INSTANCE}"
}

do_status() {
  echo "instance=${INSTANCE}"
  echo "service=${SERVICE_NAME}"
  echo "init=$(detect_init)"
  [[ -x "${INSTALL_DIR}/realm" ]] && "${INSTALL_DIR}/realm" --version || true
  if [[ -f "$PID_FILE" ]]; then
    echo "realm_pid=$(cat "$PID_FILE")"
  fi
  if cmd_exists systemctl; then systemctl status "${SERVICE_NAME}" --no-pager 2>/dev/null || true; fi
}

case "$ACTION" in
  install|update) do_install ;;
  restart) need_root; stop_instance; if [[ "$NO_SYSTEM_SERVICE" == "1" ]]; then start_noinit; else install_service; fi ;;
  uninstall) need_root; stop_instance; rm -f "/etc/systemd/system/${SERVICE_NAME}.service" "/etc/init.d/${SERVICE_NAME}" "/etc/supervisor/conf.d/${SERVICE_NAME}.conf"; rm -rf "$CONFIG_DIR"; echo "Removed instance ${INSTANCE}. Install dir and logs kept: ${INSTALL_DIR}, ${LOG_DIR}" ;;
  status) do_status ;;
esac
