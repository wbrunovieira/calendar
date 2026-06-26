---
name: deploy
description: Deploy/redeploy do Calendar em produção (VPS Contabo). Use ao fazer deploy, subir mudança pra produção, rebuildar serviço, rollback, ou verificar saúde dos containers. Cobre os playbooks Ansible E o método SSH direto (recomendado pra 1 serviço — evita queda de SSH em build longo).
---

# Deploy — Calendar (produção)

Produção: VPS **45.90.123.190** (root, `~/.ssh/id_rsa`), app em **`/opt/calendar`**, **`docker-compose.prod.yml`**, branch `main`. Serviços: calendar-core(3334), calendar-finances(3335), calendar-health(3336), agents(3337), calendar-frontend(3001/3000), finances-frontend(3003), health-frontend(3004) + langfuse(3100) + authelia(9092). Todas as portas bind em 127.0.0.1 (Nginx faz o proxy externo).

## Decisão rápida — qual usar
- **1 ou 2 serviços mudaram** (caso comum) → **SSH direto, build detached** (abaixo). Mais rápido e **não cai** em build longo.
- **Deploy completo / primeira vez** → `deploy.yml` (precisa do vault).
- **Redeploy de tudo** → `quick-deploy.yml` (rebuilda TODOS; cuidado: build longo pode derrubar SSH).
- **Voltar versão** → `rollback.yml`.

## ✅ Método recomendado: SSH direto (targeted, build detached)
Sobrevive a queda de SSH (build roda detached no servidor) e só rebuilda o que mudou.
```bash
# 1) sync + build detached (no servidor)
ssh -n -i ~/.ssh/id_rsa -o LogLevel=ERROR root@45.90.123.190 \
 "cd /opt/calendar && git pull --ff-only origin main && \
  nohup docker-compose -f docker-compose.prod.yml build calendar-finances finances-frontend \
  > /tmp/build.log 2>&1 & echo PID \$!"

# 2) esperar o build (loop no SERVIDOR, sleep remoto é ok):
ssh -n root@45.90.123.190 'P=<PID>; for i in $(seq 1 18); do kill -0 $P 2>/dev/null && sleep 10 || break; done; \
  kill -0 $P 2>/dev/null && echo RODANDO || echo BUILD-OK; \
  grep -iE "error|failed" /tmp/build.log | grep -vi "0 errors" | tail -3'

# 3) subir (SEMPRE com cd /opt/calendar) + verificar:
ssh -n root@45.90.123.190 "cd /opt/calendar && docker-compose -f docker-compose.prod.yml up -d calendar-finances finances-frontend"
ssh -n root@45.90.123.190 "curl -s -o /dev/null -w 'api %{http_code}\n' http://127.0.0.1:3335/health"
```
- Recarregar **env vars** (ex: novo `.env`) precisa **`up -d --force-recreate <svc>`** (restart simples não relê env).
- `git pull --ff-only` preserva mods locais do servidor (há edições não-commitadas em `scripts/backup.sh` e `scripts/health-check.sh`).

## Ansible
Rodar **da pasta `deploy/ansible/`**, sempre com `-i inventory/production.yml` (não há `ansible.cfg`). Os playbooks usam `raw:` (não precisa python no remoto) e `docker compose` (v2).
```bash
cd deploy/ansible
ansible-playbook -i inventory/production.yml playbooks/quick-deploy.yml          # git pull + up -d --build (TODOS) + migrate + health + cleanup
ansible-playbook -i inventory/production.yml playbooks/deploy.yml --ask-vault-pass # setup completo: escreve .env do vault, build, migrate
ansible-playbook -i inventory/production.yml playbooks/rollback.yml -e "commits=1" # git reset --hard HEAD~N + rebuild + health
```
Outros (setup único): `setup-nginx-ssl.yml`, `setup-authelia.yml`, `setup-health-check.yml`, `setup-backup.yml`, `sync-data.yml`.
- **Vault**: `deploy.yml` injeta segredos de `inventory/group_vars/all/vault.yml` (gitignored, encriptado) → `--ask-vault-pass`. quick-deploy/rollback **não** precisam de vault.
- `deploy.yml` faz **`git reset --hard origin/main`** → **descarta mods locais do servidor** (backup.sh/health-check.sh seriam perdidos). quick-deploy faz `git pull` (pode conflitar com elas). Por isso o SSH direto com `--ff-only` costuma ser mais seguro hoje.

## Migrations (Prisma, calendar-core)
Em produção é `migrate deploy` (NÃO `dev`):
```bash
ssh root@45.90.123.190 "cd /opt/calendar && docker-compose -f docker-compose.prod.yml exec -T calendar-core npx prisma migrate deploy"
```
quick-deploy/deploy já rodam isso. Só roda automaticamente se houver migration pendente.

## Health check (verificar tudo)
```bash
ssh -n root@45.90.123.190 'for u in 3334/ 3335/health 3336/api/v1/health 3337/health 3003/ 3001/ 3004/; do \
  echo -n "$u -> "; curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:$u; done; \
  docker ps --format "{{.Names}}: {{.Status}}" | grep -E "calendar|finances|health|agents|langfuse"'
```
(calendar-core não tem rota `/health` — responde 404 mas está healthy.) Sanidade de dados: ver skill `financas`.

## Gotchas (memória do projeto)
- **quick-deploy.yml cai** ("non-zero return code") em build longo (SSH dropa) → prefira **SSH direto** com build detached.
- **Colons em `raw:` não-quotado** quebram o YAML → sempre quote: `raw: "cmd || echo 'msg'"`.
- Inventory é **`production.yml`** (não `hosts.yml`).
- `up -d` rodado da pasta errada falha ("stat /root/docker-compose.prod.yml: no such file") → **sempre `cd /opt/calendar`** antes (com `;`/`&&`, não dentro de subshell em background).
- `ssh -n` em loops (senão consome stdin). Ruído SSH: `-o LogLevel=ERROR` + `2>/dev/null`.
- `docker-compose restart` NÃO relê env → use `up -d --force-recreate`.
- Limpeza pós-deploy: `docker image prune -f && docker builder prune -f` (os playbooks já fazem).
