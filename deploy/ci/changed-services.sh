#!/usr/bin/env bash
# Mapeia o diff entre o SHA deployado e o HEAD para serviços do docker-compose.prod.yml.
# Uso: changed-services.sh <sha-deployado> <override>
#   override: 'auto' (detecta pelo diff), 'all', ou lista explícita ('calendar-finances agents')
# Saída (formato GITHUB_OUTPUT):
#   services=<lista separada por espaço>
#   migrate=<true|false>  (diff toca services/calendar-core/prisma/migrations/)
set -euo pipefail

DEPLOYED="${1:-}"
OVERRIDE="${2:-auto}"

ALL_SERVICES="calendar-core calendar-finances calendar-health agents calendar-frontend finances-frontend health-frontend"

if [ "$OVERRIDE" != "auto" ] && [ "$OVERRIDE" != "all" ] && [ -n "$OVERRIDE" ]; then
  echo "services=$OVERRIDE"
  echo "migrate=false"
  exit 0
fi

if [ "$OVERRIDE" = "all" ] || [ -z "$DEPLOYED" ] || ! git cat-file -e "$DEPLOYED" 2>/dev/null; then
  # SHA desconhecido → default seguro: rebuilda todos (e assume migrations pendentes)
  echo "services=$ALL_SERVICES"
  echo "migrate=true"
  exit 0
fi

CHANGED_FILES=$(git diff --name-only "$DEPLOYED"..HEAD)

SERVICES=""
add() { case " $SERVICES " in *" $1 "*) ;; *) SERVICES="$SERVICES $1" ;; esac; }

while IFS= read -r f; do
  case "$f" in
    services/calendar-core/*)     add calendar-core ;;
    services/calendar-finances/*) add calendar-finances ;;
    services/calendar-health/*)   add calendar-health ;;
    services/agents/*)            add agents ;;
    services/calendar-frontend/*) add calendar-frontend ;;
    services/finances-frontend/*) add finances-frontend ;;
    services/health-frontend/*)   add health-frontend ;;
    docker-compose.prod.yml)      SERVICES=" $ALL_SERVICES" ;;
  esac
done <<< "$CHANGED_FILES"

MIGRATE=false
if echo "$CHANGED_FILES" | grep -q '^services/calendar-core/prisma/migrations/'; then
  MIGRATE=true
fi

echo "services=$(echo $SERVICES | xargs)"
echo "migrate=$MIGRATE"
