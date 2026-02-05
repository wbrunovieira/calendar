#!/usr/bin/env bash
# Daily server health check — sends summary via WhatsApp (Evolution API)
# If WhatsApp fails, saves full report to /var/log/health-check-failed.log
# Crontab: CRON_TZ=America/Sao_Paulo → 0 12 * * *
# Manual:  /opt/calendar/scripts/health-check.sh
set -euo pipefail

# ── Config ────────────────────────────────────────────────────
APP_DIR="${APP_DIR:-/opt/calendar}"
LOG_FILE="/var/log/health-check.log"
FAILED_LOG="/var/log/health-check-failed.log"
MAX_RETRIES=2
RETRY_DELAY=10

# Load from .env if not already set
if [[ -f "$APP_DIR/.env" ]]; then
    while IFS='=' read -r key value; do
        [[ -z "$key" || "$key" =~ ^# ]] && continue
        value="${value%%#*}"
        value="${value%"${value##*[![:space:]]}"}"
        case "$key" in
            EVOLUTION_API_URL|EVOLUTION_API_KEY|EVOLUTION_INSTANCE|WHATSAPP_NOTIFY_NUMBER)
                [[ -z "${!key:-}" ]] && export "$key=$value"
                ;;
        esac
    done < "$APP_DIR/.env"
fi

EVOLUTION_API_URL="${EVOLUTION_API_URL:-}"
EVOLUTION_API_KEY="${EVOLUTION_API_KEY:-}"
EVOLUTION_INSTANCE="${EVOLUTION_INSTANCE:-wbdigital}"
WHATSAPP_NOTIFY_NUMBER="${WHATSAPP_NOTIFY_NUMBER:-5511982864581}"

# Expected containers
EXPECTED_CONTAINERS=(
    calendar-core
    calendar-frontend
    finances-frontend
    health-frontend
    calendar-finances
    calendar-health
    calendar-agents
    postgres
    langfuse-web
    langfuse-worker
    langfuse-postgres
    langfuse-clickhouse
    langfuse-minio
    langfuse-redis
    authelia
)

# Endpoints: "label|url|expected_http_code"
ENDPOINTS=(
    "Calendar API|https://calendar-api.wbdigitalsolutions.com/|200"
    "Finances API|https://calendar-finances.wbdigitalsolutions.com/api/v1/health|200"
    "Health API|https://calendar-health.wbdigitalsolutions.com/api/v1/health|200"
    "Agents API|https://calendar-agents.wbdigitalsolutions.com/health|200"
    "Calendar UI|https://calendar.wbdigitalsolutions.com/|302"
    "Finances UI|https://finances.wbdigitalsolutions.com/|302"
    "Health UI|https://health.wbdigitalsolutions.com/|302"
    "Langfuse|https://calendar-langfuse.wbdigitalsolutions.com/|302"
    "Authelia|http://127.0.0.1:9092/api/health|200"
)

# SSL domains to check
SSL_DOMAINS=(
    calendar.wbdigitalsolutions.com
    finances.wbdigitalsolutions.com
    health.wbdigitalsolutions.com
    calendar-api.wbdigitalsolutions.com
    calendar-finances.wbdigitalsolutions.com
    calendar-health.wbdigitalsolutions.com
    calendar-agents.wbdigitalsolutions.com
    calendar-langfuse.wbdigitalsolutions.com
    auth.wbdigitalsolutions.com
)

# ── Helpers ───────────────────────────────────────────────────
NOW=$(date '+%d/%m/%Y %H:%M')
ALERTS=()
WARNINGS=()

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" >> "$LOG_FILE"; }
add_alert()   { ALERTS+=("$1"); }
add_warning() { WARNINGS+=("$1"); }

# ── 1. Host Metrics ──────────────────────────────────────────
collect_host() {
    UPTIME_DAYS=$(awk '{print int($1/86400)}' /proc/uptime 2>/dev/null || echo "?")
    LOAD=$(awk '{printf "%s | %s | %s", $1, $2, $3}' /proc/loadavg 2>/dev/null || echo "?")

    RAM_TOTAL_KB=$(grep MemTotal /proc/meminfo | awk '{print $2}')
    RAM_AVAIL_KB=$(grep MemAvailable /proc/meminfo | awk '{print $2}')
    RAM_USED_KB=$((RAM_TOTAL_KB - RAM_AVAIL_KB))
    RAM_TOTAL_GB=$(awk "BEGIN {printf \"%.1f\", $RAM_TOTAL_KB/1048576}")
    RAM_USED_GB=$(awk "BEGIN {printf \"%.1f\", $RAM_USED_KB/1048576}")
    RAM_PCT=$(awk "BEGIN {printf \"%.0f\", $RAM_USED_KB/$RAM_TOTAL_KB*100}")

    DISK_LINE=$(df -BG / | tail -1)
    DISK_TOTAL=$(echo "$DISK_LINE" | awk '{gsub("G",""); print $2}')
    DISK_USED=$(echo "$DISK_LINE" | awk '{gsub("G",""); print $3}')
    DISK_PCT=$(echo "$DISK_LINE" | awk '{gsub("%",""); print $5}')

    [[ "$RAM_PCT" -ge 90 ]] && add_alert "RAM: ${RAM_USED_GB}/${RAM_TOTAL_GB} GB (${RAM_PCT}%)"
    [[ "$RAM_PCT" -ge 80 && "$RAM_PCT" -lt 90 ]] && add_warning "RAM: ${RAM_USED_GB}/${RAM_TOTAL_GB} GB (${RAM_PCT}%)"
    [[ "$DISK_PCT" -ge 85 ]] && add_alert "Disco: ${DISK_USED}/${DISK_TOTAL} GB (${DISK_PCT}%)"
    [[ "$DISK_PCT" -ge 75 && "$DISK_PCT" -lt 85 ]] && add_warning "Disco: ${DISK_USED}/${DISK_TOTAL} GB (${DISK_PCT}%)"

    HOST_SECTION="CPU: ${LOAD}
RAM: ${RAM_USED_GB}/${RAM_TOTAL_GB} GB (${RAM_PCT}%)
Disco: ${DISK_USED}/${DISK_TOTAL} GB (${DISK_PCT}%)
Uptime: ${UPTIME_DAYS} dias"
}

# ── 2. Docker Containers ─────────────────────────────────────
collect_containers() {
    RUNNING=0
    TOTAL=${#EXPECTED_CONTAINERS[@]}
    CONTAINER_ISSUES=""

    STATS=$(docker stats --no-stream --format "{{.Name}}|{{.MemUsage}}" 2>/dev/null || echo "")

    for name in "${EXPECTED_CONTAINERS[@]}"; do
        STATUS=$(docker inspect --format='{{.State.Status}}' "$name" 2>/dev/null || echo "not_found")
        RESTARTS=$(docker inspect --format='{{.RestartCount}}' "$name" 2>/dev/null || echo "0")

        if [[ "$STATUS" == "running" ]]; then
            RUNNING=$((RUNNING + 1))

            MEM_LIMIT_BYTES=$(docker inspect --format='{{.HostConfig.Memory}}' "$name" 2>/dev/null || echo "0")
            if [[ "$MEM_LIMIT_BYTES" -gt 0 ]]; then
                MEM_LIMIT_MB=$((MEM_LIMIT_BYTES / 1048576))
                MEM_RAW=$(echo "$STATS" | grep "^${name}|" | cut -d'|' -f2 | awk -F'/' '{print $1}' | xargs || echo "")

                if [[ -n "$MEM_RAW" ]]; then
                    MEM_NUM=$(echo "$MEM_RAW" | sed 's/[^0-9.]//g')
                    MEM_UNIT=$(echo "$MEM_RAW" | sed 's/[0-9. ]//g')

                    case "$MEM_UNIT" in
                        GiB) MEM_USED_MB=$(awk "BEGIN {printf \"%.0f\", $MEM_NUM * 1024}") ;;
                        MiB) MEM_USED_MB=$(awk "BEGIN {printf \"%.0f\", $MEM_NUM}") ;;
                        KiB) MEM_USED_MB=$(awk "BEGIN {printf \"%.0f\", $MEM_NUM / 1024}") ;;
                        *)   MEM_USED_MB=$(awk "BEGIN {printf \"%.0f\", $MEM_NUM}") ;;
                    esac

                    LOCAL_PCT=$(awk "BEGIN {printf \"%.0f\", $MEM_USED_MB/$MEM_LIMIT_MB*100}")

                    if [[ "$LOCAL_PCT" -ge 90 ]]; then
                        add_alert "${name}: ${MEM_USED_MB}/${MEM_LIMIT_MB} MB (${LOCAL_PCT}%)"
                    elif [[ "$LOCAL_PCT" -ge 80 ]]; then
                        add_warning "${name}: ${MEM_USED_MB}/${MEM_LIMIT_MB} MB (${LOCAL_PCT}%)"
                    fi
                fi
            fi

            [[ "$RESTARTS" -gt 0 ]] && add_warning "${name}: ${RESTARTS} restarts"
        else
            add_alert "${name}: ${STATUS}"
            CONTAINER_ISSUES="${CONTAINER_ISSUES}
  - ${name}: ${STATUS}"
        fi
    done

    if [[ "$RUNNING" -eq "$TOTAL" ]]; then
        CONTAINER_SECTION="Containers: ${RUNNING}/${TOTAL} running"
    else
        CONTAINER_SECTION="Containers: ${RUNNING}/${TOTAL} running${CONTAINER_ISSUES}"
    fi
}

# ── 3. HTTP Endpoints ─────────────────────────────────────────
check_endpoints() {
    EP_OK=0
    EP_TOTAL=${#ENDPOINTS[@]}
    EP_ISSUES=""

    for entry in "${ENDPOINTS[@]}"; do
        IFS='|' read -r label url expected <<< "$entry"
        CODE=$(curl -sf -o /dev/null -w '%{http_code}' --max-time 10 "$url" 2>/dev/null || echo "000")

        if [[ "$CODE" == "$expected" ]]; then
            EP_OK=$((EP_OK + 1))
        else
            add_alert "${label}: HTTP ${CODE} (esperado ${expected})"
            EP_ISSUES="${EP_ISSUES}
  - ${label}: ${CODE}"
        fi
    done

    if [[ "$EP_OK" -eq "$EP_TOTAL" ]]; then
        ENDPOINT_SECTION="Endpoints: ${EP_OK}/${EP_TOTAL} OK"
    else
        ENDPOINT_SECTION="Endpoints: ${EP_OK}/${EP_TOTAL} OK${EP_ISSUES}"
    fi
}

# ── 4. SSL Certificate Expiry ─────────────────────────────────
check_ssl() {
    SSL_MIN_DAYS=999

    for domain in "${SSL_DOMAINS[@]}"; do
        EXPIRY=$(echo | openssl s_client -servername "$domain" -connect "$domain":443 2>/dev/null \
            | openssl x509 -noout -enddate 2>/dev/null | cut -d= -f2)

        if [[ -n "$EXPIRY" ]]; then
            EXPIRY_EPOCH=$(date -d "$EXPIRY" +%s 2>/dev/null || echo "0")
            NOW_EPOCH=$(date +%s)
            DAYS_LEFT=$(( (EXPIRY_EPOCH - NOW_EPOCH) / 86400 ))

            [[ "$DAYS_LEFT" -lt "$SSL_MIN_DAYS" ]] && SSL_MIN_DAYS=$DAYS_LEFT

            if [[ "$DAYS_LEFT" -lt 7 ]]; then
                add_alert "SSL ${domain}: ${DAYS_LEFT} dias"
            elif [[ "$DAYS_LEFT" -lt 14 ]]; then
                add_warning "SSL ${domain}: ${DAYS_LEFT} dias"
            fi
        else
            add_warning "SSL ${domain}: falha ao verificar"
        fi
    done

    if [[ "$SSL_MIN_DAYS" -lt 999 ]]; then
        SSL_SECTION="SSL: min ${SSL_MIN_DAYS} dias"
    else
        SSL_SECTION="SSL: n/a"
    fi
}

# ── 5. Database Checks ────────────────────────────────────────
check_databases() {
    CAL_DB_SIZE=$(docker exec postgres psql -U calendar -d calendar_db -t -c \
        "SELECT pg_size_pretty(pg_database_size('calendar_db'));" 2>/dev/null | xargs || echo "ERRO")

    LF_DB_SIZE=$(docker exec langfuse-postgres psql -U langfuse -d langfuse_db -t -c \
        "SELECT pg_size_pretty(pg_database_size('langfuse_db'));" 2>/dev/null | xargs || echo "ERRO")

    [[ "$CAL_DB_SIZE" == "ERRO" ]] && add_alert "Calendar DB: conexao falhou"
    [[ "$LF_DB_SIZE" == "ERRO" ]] && add_alert "Langfuse DB: conexao falhou"

    DB_SECTION="DB: calendar ${CAL_DB_SIZE} | langfuse ${LF_DB_SIZE}"
}

# ── 6. Format Message ─────────────────────────────────────────
format_message() {
    local has_alerts=${#ALERTS[@]}
    local has_warnings=${#WARNINGS[@]}

    if [[ "$has_alerts" -gt 0 ]]; then
        HEADER="Server Health — ALERTA — ${NOW}"
    else
        HEADER="Server Health — ${NOW}"
    fi

    MSG="${HEADER}

${HOST_SECTION}

${CONTAINER_SECTION}

${ENDPOINT_SECTION}

${SSL_SECTION}

${DB_SECTION}"

    if [[ "$has_alerts" -gt 0 ]]; then
        MSG="${MSG}

ALERTAS:"
        for a in "${ALERTS[@]}"; do
            MSG="${MSG}
  - ${a}"
        done
    fi

    if [[ "$has_warnings" -gt 0 ]]; then
        MSG="${MSG}

AVISOS:"
        for w in "${WARNINGS[@]}"; do
            MSG="${MSG}
  - ${w}"
        done
    fi

    if [[ "$has_alerts" -eq 0 && "$has_warnings" -eq 0 ]]; then
        MSG="${MSG}

Tudo OK"
    fi
}

# ── 7. Send WhatsApp (with retry) ────────────────────────────
send_whatsapp() {
    if [[ -z "$EVOLUTION_API_URL" || -z "$EVOLUTION_API_KEY" || -z "$WHATSAPP_NOTIFY_NUMBER" ]]; then
        log "WARN: WhatsApp nao configurado (EVOLUTION_API_URL, EVOLUTION_API_KEY ou WHATSAPP_NOTIFY_NUMBER vazio)"
        SEND_ERROR="variaveis nao configuradas"
        return 1
    fi

    local payload
    payload=$(jq -n --arg number "$WHATSAPP_NOTIFY_NUMBER" --arg text "$MSG" \
        '{number: $number, text: $text}')

    local attempt=1
    SEND_ERROR=""

    while [[ "$attempt" -le "$MAX_RETRIES" ]]; do
        local response_body
        response_body=$(mktemp)

        local http_code
        http_code=$(curl -s -o "$response_body" -w '%{http_code}' --max-time 15 \
            -X POST "${EVOLUTION_API_URL}/message/sendText/${EVOLUTION_INSTANCE}" \
            -H "apikey: ${EVOLUTION_API_KEY}" \
            -H "Content-Type: application/json" \
            -d "$payload" 2>/dev/null || echo "000")

        if [[ "$http_code" == "201" || "$http_code" == "200" ]]; then
            log "WhatsApp enviado com sucesso (tentativa ${attempt})"
            rm -f "$response_body"
            return 0
        fi

        SEND_ERROR="HTTP ${http_code}: $(cat "$response_body" 2>/dev/null | head -c 300 || echo "sem resposta")"
        rm -f "$response_body"

        log "WARN: WhatsApp tentativa ${attempt}/${MAX_RETRIES} falhou — ${SEND_ERROR}"

        if [[ "$attempt" -lt "$MAX_RETRIES" ]]; then
            sleep "$RETRY_DELAY"
        fi

        attempt=$((attempt + 1))
    done

    log "ERRO: WhatsApp falhou apos ${MAX_RETRIES} tentativas"
    return 1
}

# ── 8. Save failure report ────────────────────────────────────
save_failure_report() {
    {
        echo "============================================"
        echo "  HEALTH CHECK — FALHA NO ENVIO"
        echo "  $(date '+%Y-%m-%d %H:%M:%S')"
        echo "============================================"
        echo ""
        echo "Motivo: ${SEND_ERROR:-desconhecido}"
        echo "Tentativas: ${MAX_RETRIES}"
        echo "Destino: ${WHATSAPP_NOTIFY_NUMBER}"
        echo "API: ${EVOLUTION_API_URL}/message/sendText/${EVOLUTION_INSTANCE}"
        echo ""
        echo "── Mensagem que seria enviada ──────────────"
        echo ""
        echo "$MSG"
        echo ""
        echo "── Fim ─────────────────────────────────────"
        echo ""
    } >> "$FAILED_LOG"

    log "Relatorio de falha salvo em ${FAILED_LOG}"
}

# ── 9. Deliver Message ────────────────────────────────────────
deliver() {
    if send_whatsapp; then
        return 0
    fi

    save_failure_report
    return 1
}

# ── Main ──────────────────────────────────────────────────────
main() {
    log "=== Health check iniciado ==="

    collect_host
    collect_containers
    check_endpoints
    check_ssl
    check_databases
    format_message

    # Always log locally
    echo "$MSG" >> "$LOG_FILE"
    echo "---" >> "$LOG_FILE"

    # Print to stdout (manual runs)
    echo "$MSG"

    # Deliver via WhatsApp, save report on failure
    if deliver; then
        log "Health check concluido (whatsapp)"
    else
        log "Health check concluido (falha no envio — ver ${FAILED_LOG})"
    fi
}

main "$@"
