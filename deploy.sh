#!/usr/bin/env bash
# OpsLog 统一部署入口（对齐 cloudsystem/auth all_in_one.sh 体验，面向 Docker）
#
# 能力：build / up / down / restart / logs / status / ps / pull / config / run / clean / help
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

PROJECT_NAME="${PROJECT_NAME:-opslog}"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml}"
CONFIG_FILE="${CONFIG_FILE:-configs/opslog.yml}"
CONSOLE_URL="${CONSOLE_URL:-http://127.0.0.1:8600/}"

log(){ echo "[$(date '+%H:%M:%S')] $*"; }
die(){ log "ERROR: $*"; exit 1; }

_DESC_COL=34
_u_cmd_sep=0
_u_cmd(){
  if [[ "${_u_cmd_sep}" -eq 1 ]]; then
    echo
  fi
  _u_cmd_sep=1
  printf " %-$(( _DESC_COL ))s %s\n" "$1" "$2"
}
_u_opt(){ printf "  %-$(( _DESC_COL - 1 ))s %s\n" "$1" "$2"; }
_u_group(){
  echo
  if [[ -n "${2:-}" ]]; then
    printf "%s: %s\n" "$1" "$2"
  else
    printf "%s:\n" "$1"
  fi
}
_u_envs(){ echo; printf " env:\n"; }
_u_opts(){ echo; printf " options:\n"; }
_u_cmds(){ _u_cmd_sep=0; echo; }

load_dotenv(){
  local envfile="$ROOT/.env"
  [[ -f "$envfile" ]] || return 0
  set -a
  # shellcheck disable=SC1090
  source "$envfile"
  set +a
}

compose(){
  if docker compose version >/dev/null 2>&1; then
    docker compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" "$@"
  elif command -v docker-compose >/dev/null 2>&1; then
    docker-compose -p "$PROJECT_NAME" -f "$COMPOSE_FILE" "$@"
  else
    die "需要 Docker Compose（docker compose 或 docker-compose）"
  fi
}

usage(){
  cat <<EOF
OpsLog 统一部署入口

用法:
 ./deploy.sh <命令> [参数...]
 ./deploy.sh <命令> -h,--help
EOF

  if [[ -f "$ROOT/.env" ]]; then
    echo
    echo "提示: 已加载 .env"
    echo "  路径: $ROOT/.env"
  fi

  _u_group "docker" "镜像与编排（默认）"
  _u_envs
  _u_opt "PROJECT_NAME"            "Compose 项目名（默认 opslog）"
  _u_opt "COMPOSE_FILE"            "Compose 文件（默认 docker-compose.yml）"
  _u_opt "CONFIG_FILE"             "挂载配置（默认 configs/opslog.yml）"
  _u_opt "CONSOLE_URL"             "控制台 URL（默认 http://127.0.0.1:8600/）"
  _u_opt "COMPOSE_PROFILES"        "可选 profiles：mysql / clickhouse / full"
  _u_cmds
  _u_cmd "build"                   "构建 opslog 镜像"
  _u_cmd "up [-d]"                 "启动服务（默认 -d 后台）"
  _u_cmd "down"                    "停止并移除容器"
  _u_cmd "restart"                 "重启服务"
  _u_cmd "logs [-f] [svc]"         "查看日志（默认 opslog，-f 跟随）"
  _u_cmd "status / ps"             "容器状态"
  _u_cmd "pull"                    "拉取依赖镜像"
  _u_cmd "config"                  "校验并打印 compose 配置"
  _u_cmd "url"                     "打印控制台与端口说明"

  _u_group "local" "本机开发（不经过 Docker）"
  _u_cmds
  _u_cmd "run"                     "go run ./cmd/opslog-server -config ..."
  _u_cmd "test"                    "go test ./..."
  _u_cmd "tidy"                    "go mod tidy"
  _u_cmd "clean"                   "清理本地 data/ 与构建缓存提示"

  _u_group "help"
  _u_cmds
  _u_cmd "help"                    "显示本帮助"
  echo
  cat <<EOF
端口:
  8600/tcp   Web Console + Query API + HTTP/WS ingest
  8141/tcp   TCP ingest
  8900/tcp   gRPC ingest

示例:
  ./deploy.sh build
  ./deploy.sh up
  ./deploy.sh logs -f
  ./deploy.sh url
  COMPOSE_PROFILES=full ./deploy.sh up
  ./deploy.sh run
EOF
}

usage_up(){
  cat <<EOF
up

用法:
 ./deploy.sh up [-d|--detach] [--build] [--profile NAME]

options:
 -d,--detach                    后台运行（默认开启）
 --foreground                   前台运行
 --build                        启动前重建镜像
 --profile <name>               启用 compose profile（可多次；或用 COMPOSE_PROFILES）
EOF
}

cmd_build(){
  log "build image..."
  compose build "$@"
  log "build done"
}

cmd_up(){
  local detach=1
  local build=0
  local -a profiles=()
  local -a pass=()
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -h|--help) usage_up; return 0 ;;
      -d|--detach) detach=1; shift ;;
      --foreground) detach=0; shift ;;
      --build) build=1; shift ;;
      --profile) profiles+=("$2"); shift 2 ;;
      *) pass+=("$1"); shift ;;
    esac
  done
  if [[ -n "${COMPOSE_PROFILES:-}" ]]; then
    IFS=',' read -r -a _prof <<< "$COMPOSE_PROFILES"
    profiles+=("${_prof[@]}")
  fi
  local -a args=()
  for p in "${profiles[@]+"${profiles[@]}"}"; do
    [[ -n "$p" ]] || continue
    args+=(--profile "$p")
  done
  if [[ "$build" -eq 1 ]]; then
    compose "${args[@]+"${args[@]}"}" build
  fi
  if [[ "$detach" -eq 1 ]]; then
    compose "${args[@]+"${args[@]}"}" up -d "${pass[@]+"${pass[@]}"}"
  else
    compose "${args[@]+"${args[@]}"}" up "${pass[@]+"${pass[@]}"}"
  fi
  log "console: $CONSOLE_URL"
}

cmd_down(){
  compose down "$@"
}

cmd_restart(){
  compose restart "$@"
}

cmd_logs(){
  local follow=0
  local svc="opslog"
  local -a pass=()
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -f|--follow) follow=1; shift ;;
      -h|--help)
        echo "usage: ./deploy.sh logs [-f] [service]"
        return 0
        ;;
      *)
        if [[ "$svc" == "opslog" && "$1" != -* && ${#pass[@]} -eq 0 ]]; then
          svc="$1"
        else
          pass+=("$1")
        fi
        shift
        ;;
    esac
  done
  if [[ "$follow" -eq 1 ]]; then
    compose logs -f "$svc" "${pass[@]+"${pass[@]}"}"
  else
    compose logs --tail=200 "$svc" "${pass[@]+"${pass[@]}"}"
  fi
}

cmd_status(){
  compose ps "$@"
}

cmd_pull(){
  compose pull "$@"
}

cmd_config(){
  compose config "$@"
}

cmd_url(){
  cat <<EOF
OpsLog endpoints

 Console:  $CONSOLE_URL
 Health:   ${CONSOLE_URL}api/health
 Ingest:   POST ${CONSOLE_URL}ingest
 Stream:   WS   ${CONSOLE_URL}stream
 TCP:      tcp://127.0.0.1:8141
 gRPC:     127.0.0.1:8900
 Data vol: docker volume / compose opslog-data (./data when local run)

SDK tip:
  agent.WithEndpoint("http://127.0.0.1:8600/ingest")
EOF
}

cmd_run(){
  local cfg="${1:-$CONFIG_FILE}"
  log "go run ./cmd/opslog-server -config $cfg"
  go run ./cmd/opslog-server -config "$cfg"
}

cmd_test(){
  go test ./...
}

cmd_tidy(){
  go mod tidy
}

cmd_clean(){
  log "remove ./data (local filesystem output)"
  rm -rf "$ROOT/data"
  log "optional: docker builder prune / compose down -v"
}

load_dotenv

CMD="${1:-help}"
shift || true

case "$CMD" in
  help|-h|--help) usage ;;
  build) cmd_build "$@" ;;
  up) cmd_up "$@" ;;
  down) cmd_down "$@" ;;
  restart) cmd_restart "$@" ;;
  logs|log) cmd_logs "$@" ;;
  status|ps) cmd_status "$@" ;;
  pull) cmd_pull "$@" ;;
  config) cmd_config "$@" ;;
  url) cmd_url "$@" ;;
  run) cmd_run "$@" ;;
  test) cmd_test "$@" ;;
  tidy) cmd_tidy "$@" ;;
  clean) cmd_clean "$@" ;;
  *) die "未知命令: ${CMD}（./deploy.sh help）" ;;
esac
