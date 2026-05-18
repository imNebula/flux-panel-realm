#!/usr/bin/env bash
set -euo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml}"
PURGE=0
REMOVE_IMAGES=0
REMOVE_FILES=0
FORCE=0

usage() {
  cat <<EOF
Usage: $0 [options]

Quickly uninstall the Flux Realm panel Docker deployment.

Options:
  --compose-file FILE   Compose file path. Default: docker-compose.yml
  --purge               Remove named data volumes mysql_data and backend_logs
  --remove-images       Remove panel Docker images after stopping containers
  --remove-files        Remove docker-compose.yml, realm.sql, and .env
  --all                 Same as --purge --remove-images --remove-files
  -y, --yes             Do not prompt for confirmation
  -h, --help            Show this help

Default behavior keeps database/log volumes and local config files.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --compose-file)
      COMPOSE_FILE="$2"
      shift 2
      ;;
    --purge)
      PURGE=1
      shift
      ;;
    --remove-images)
      REMOVE_IMAGES=1
      shift
      ;;
    --remove-files)
      REMOVE_FILES=1
      shift
      ;;
    --all)
      PURGE=1
      REMOVE_IMAGES=1
      REMOVE_FILES=1
      shift
      ;;
    -y|--yes)
      FORCE=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage
      exit 1
      ;;
  esac
done

cmd_exists() {
  command -v "$1" >/dev/null 2>&1
}

detect_docker_compose() {
  if cmd_exists docker-compose; then
    DOCKER_COMPOSE_CMD=(docker-compose)
  elif cmd_exists docker && docker compose version >/dev/null 2>&1; then
    DOCKER_COMPOSE_CMD=(docker compose)
  else
    DOCKER_COMPOSE_CMD=()
  fi
}

confirm() {
  if [[ "$FORCE" == "1" ]]; then
    return 0
  fi

  echo "This will stop and remove Flux Realm panel containers."
  if [[ "$PURGE" == "1" ]]; then
    echo "Data volumes mysql_data and backend_logs will also be removed."
  else
    echo "Data volumes will be kept. Use --purge to remove them."
  fi

  read -r -p "Continue? (y/N): " answer
  [[ "$answer" == "y" || "$answer" == "Y" ]]
}

remove_known_containers() {
  if ! cmd_exists docker; then
    echo "docker is required." >&2
    exit 1
  fi

  docker rm -f flux-realm-mysql springboot-backend vite-frontend >/dev/null 2>&1 || true
  docker network rm flux-realm-network >/dev/null 2>&1 || true
}

remove_known_volumes() {
  docker volume rm mysql_data backend_logs >/dev/null 2>&1 || true
}

remove_known_images() {
  local images
  images="$(docker images --format '{{.Repository}}:{{.Tag}}' \
    | grep -E '(^|/)flux-panel-realm/(springboot-backend|vite-frontend):' || true)"

  if [[ -n "$images" ]]; then
    while IFS= read -r image; do
      [[ -n "$image" ]] || continue
      docker rmi -f "$image" >/dev/null 2>&1 || true
    done <<<"$images"
  fi
}

remove_local_files() {
  rm -f "$COMPOSE_FILE" realm.sql .env
}

main() {
  detect_docker_compose

  if ! confirm; then
    echo "Cancelled."
    exit 0
  fi

  if [[ -f "$COMPOSE_FILE" && "${#DOCKER_COMPOSE_CMD[@]}" -gt 0 ]]; then
    down_args=(-f "$COMPOSE_FILE" down --remove-orphans)
    if [[ "$PURGE" == "1" ]]; then
      down_args+=(--volumes)
    fi
    if [[ "$REMOVE_IMAGES" == "1" ]]; then
      down_args+=(--rmi all)
    fi
    "${DOCKER_COMPOSE_CMD[@]}" "${down_args[@]}"
  else
    remove_known_containers
    if [[ "$PURGE" == "1" ]]; then
      remove_known_volumes
    fi
    if [[ "$REMOVE_IMAGES" == "1" ]]; then
      remove_known_images
    fi
  fi

  if [[ "$REMOVE_FILES" == "1" ]]; then
    remove_local_files
  fi

  echo "Uninstall complete."
  if [[ "$PURGE" != "1" ]]; then
    echo "Kept data volumes: mysql_data, backend_logs"
  fi
}

main
