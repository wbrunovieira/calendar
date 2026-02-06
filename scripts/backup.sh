#!/bin/bash
set -e

# Backup all Calendar project databases + config to Google Drive
# Crontab: 0 4 * * * /opt/calendar/scripts/backup.sh >> /var/log/calendar-backup.log 2>&1

BACKUP_DIR="/opt/backups/calendar"
APP_DIR="/opt/calendar"
GDRIVE_REMOTE="gdrive"
GDRIVE_FOLDER="backups/calendar"
RETENTION_DAYS=7
DATE=$(date +%Y-%m-%d_%H-%M-%S)
BACKUP_NAME="calendar-backup-${DATE}"
LOG_FILE="/var/log/calendar-backup.log"

# Databases to backup: "label|container|user|dbname"
DATABASES=(
    "calendar|calendar-postgres|calendar|calendar_db"
    "langfuse|langfuse-postgres|langfuse|langfuse_db"
)

# Load WhatsApp config for notifications
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

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

notify_whatsapp() {
    local msg="$1"
    [[ -z "$EVOLUTION_API_URL" || -z "$EVOLUTION_API_KEY" ]] && return 0
    local payload
    payload=$(jq -n --arg number "$WHATSAPP_NOTIFY_NUMBER" --arg text "$msg" \
        '{number: $number, text: $text}')
    curl -s -o /dev/null --max-time 10 \
        -X POST "${EVOLUTION_API_URL}/message/sendText/${EVOLUTION_INSTANCE}" \
        -H "apikey: ${EVOLUTION_API_KEY}" \
        -H "Content-Type: application/json" \
        -d "$payload" 2>/dev/null || true
}

main() {
    log "=========================================="
    log "Backup iniciado: ${BACKUP_NAME}"
    log "=========================================="

    mkdir -p "$BACKUP_DIR"
    TEMP_DIR="${BACKUP_DIR}/${BACKUP_NAME}"
    mkdir -p "$TEMP_DIR"

    ERRORS=""
    DB_SIZES=""

    # Backup each database
    for entry in "${DATABASES[@]}"; do
        IFS='|' read -r label container user dbname <<< "$entry"
        log "Banco ${label} (${container})..."

        if docker exec "$container" pg_dump -U "$user" -d "$dbname" -F c -f /tmp/database.dump 2>/dev/null; then
            docker cp "${container}:/tmp/database.dump" "${TEMP_DIR}/${label}.dump"
            docker exec "$container" rm /tmp/database.dump
            DUMP_SIZE=$(du -h "${TEMP_DIR}/${label}.dump" | cut -f1)
            log "  ${label}: ${DUMP_SIZE}"
            DB_SIZES="${DB_SIZES}${label}: ${DUMP_SIZE}, "
        else
            log "  ERRO: ${label} falhou!"
            ERRORS="${ERRORS}${label}, "
        fi
    done

    # Backup .env (sem secrets no nome do arquivo)
    if [[ -f "${APP_DIR}/.env" ]]; then
        cp "${APP_DIR}/.env" "${TEMP_DIR}/env.backup"
        log ".env copiado"
    fi

    # Backup Nginx configs
    mkdir -p "${TEMP_DIR}/nginx"
    cp /etc/nginx/sites-available/*.conf "${TEMP_DIR}/nginx/" 2>/dev/null || true
    cp /etc/nginx/sites-available/*wbdigitalsolutions* "${TEMP_DIR}/nginx/" 2>/dev/null || true
    log "Nginx configs copiados"

    # Backup Authelia config
    if [[ -d "${APP_DIR}/config/authelia" ]]; then
        mkdir -p "${TEMP_DIR}/authelia"
        cp "${APP_DIR}/config/authelia/"*.yml "${TEMP_DIR}/authelia/" 2>/dev/null || true
        log "Authelia config copiado"
    fi

    # Compress
    log "Compactando..."
    cd "$BACKUP_DIR"
    tar -czf "${BACKUP_NAME}.tar.gz" "${BACKUP_NAME}"
    rm -rf "$TEMP_DIR"

    ARCHIVE_SIZE=$(du -h "${BACKUP_DIR}/${BACKUP_NAME}.tar.gz" | cut -f1)
    log "Arquivo: ${BACKUP_NAME}.tar.gz (${ARCHIVE_SIZE})"

    # Upload to Google Drive
    log "Enviando para Google Drive..."
    if rclone listremotes 2>/dev/null | grep -q "${GDRIVE_REMOTE}:"; then
        if rclone copy "${BACKUP_DIR}/${BACKUP_NAME}.tar.gz" "${GDRIVE_REMOTE}:${GDRIVE_FOLDER}/" 2>&1; then
            log "Upload concluido"
        else
            log "ERRO: Upload para Google Drive falhou!"
            ERRORS="${ERRORS}upload, "
        fi
    else
        log "AVISO: Google Drive nao configurado"
        ERRORS="${ERRORS}gdrive nao configurado, "
    fi

    # Cleanup old backups
    log "Limpando backups antigos (>${RETENTION_DAYS} dias)..."
    find "$BACKUP_DIR" -name "calendar-backup-*.tar.gz" -mtime +${RETENTION_DAYS} -delete 2>/dev/null || true
    rclone delete "${GDRIVE_REMOTE}:${GDRIVE_FOLDER}" --min-age ${RETENTION_DAYS}d 2>/dev/null || true

    # Summary
    DB_SIZES="${DB_SIZES%, }"
    log "=========================================="
    if [[ -n "$ERRORS" ]]; then
        ERRORS="${ERRORS%, }"
        log "Backup com ERROS: ${ERRORS}"
        log "=========================================="
        notify_whatsapp "Backup Calendar — ERRO

${ERRORS}

Bancos: ${DB_SIZES}
Tamanho: ${ARCHIVE_SIZE}

Verifique: /var/log/calendar-backup.log"
    else
        log "Backup concluido com sucesso!"
        log "Local: ${BACKUP_DIR}/${BACKUP_NAME}.tar.gz"
        log "Drive: ${GDRIVE_REMOTE}:${GDRIVE_FOLDER}/${BACKUP_NAME}.tar.gz"
        log "Tamanho: ${ARCHIVE_SIZE}"
        log "=========================================="
    fi
}

main "$@"
