---
name: deploy
description: Deploy/redeploy Calendar to production (Contabo VPS). Use when deploying, shipping a change to production, rebuilding a service, rolling back, or checking container health. Covers both the Ansible playbooks AND the direct-SSH method (recommended for a single service — avoids SSH dropping during a long build).
---

# Deploy — Calendar (production)

Production: VPS **45.90.123.190** (root, `~/.ssh/id_rsa`), app at **`/opt/calendar`**, **`docker-compose.prod.yml`**, branch `main`. Services: calendar-core(3334), calendar-finances(3335), calendar-health(3336), agents(3337), calendar-frontend(3001/3000), finances-frontend(3003), health-frontend(3004) plus langfuse(3100) and authelia(9092). Every port binds to 127.0.0.1 (Nginx handles the external proxy).

## Quick decision — which one to use
- **✅ Default path (since 2026-07):** commit + push to `main` → **GitHub Actions CI/CD** (`.github/workflows/ci.yml` + `deploy.yml`). CD waits for a green CI, requires **Bruno's manual approval on the GitHub page** (environment `production` — NEVER approve via the API), and deploys over SSH as the `deploy` user, touching only the services that changed (`deploy/ci/changed-services.sh`), with a conditional migrate and a health check. Follow along with `gh run list` / `gh run watch`. Manual re-deploy: workflow_dispatch on the CD (input `services`: 'auto'|'all'|list).
- **Manual hotfix without the pipeline** (emergency) → **direct SSH, detached build** (below).
- **Full deploy / first time** → Ansible's `deploy.yml` (needs the vault).
- **Redeploy everything** → `quick-deploy.yml` (rebuilds ALL; careful, a long build can drop SSH).
- **Roll back a version** → `rollback.yml`.

## ✅ Recommended method: direct SSH (targeted, detached build)
Survives an SSH drop (the build runs detached on the server) and only rebuilds what changed.
```bash
# 1) sync + detached build (on the server)
ssh -n -i ~/.ssh/id_rsa -o LogLevel=ERROR root@45.90.123.190 \
 "cd /opt/calendar && git pull --ff-only origin main && \
  nohup docker-compose -f docker-compose.prod.yml build calendar-finances finances-frontend \
  > /tmp/build.log 2>&1 & echo PID \$!"

# 2) wait for the build (loop on the SERVER, a remote sleep is fine):
ssh -n root@45.90.123.190 'P=<PID>; for i in $(seq 1 18); do kill -0 $P 2>/dev/null && sleep 10 || break; done; \
  kill -0 $P 2>/dev/null && echo RUNNING || echo BUILD-OK; \
  grep -iE "error|failed" /tmp/build.log | grep -vi "0 errors" | tail -3'

# 3) bring it up (ALWAYS with cd /opt/calendar) + verify:
ssh -n root@45.90.123.190 "cd /opt/calendar && docker-compose -f docker-compose.prod.yml up -d calendar-finances finances-frontend"
ssh -n root@45.90.123.190 "curl -s -o /dev/null -w 'api %{http_code}\n' http://127.0.0.1:3335/health"
```
- Reloading **env vars** (e.g. a new `.env`) requires **`up -d --force-recreate <svc>`** (a plain restart does not re-read the env).
- `git pull --ff-only` preserves the server's local modifications (there are uncommitted edits in `scripts/backup.sh` and `scripts/health-check.sh`).

## Ansible
There is an `ansible.cfg` (in `deploy/ansible/`) already setting `inventory = inventory/production.yml`, `remote_user=root`, `host_key_checking=False`, `become=True`. So **run from the `deploy/ansible/` folder** and `-i` is unnecessary. The playbooks use `raw:` (no python needed on the remote) and `docker compose` (v2).
```bash
cd deploy/ansible
ansible-playbook playbooks/quick-deploy.yml            # git pull + up -d --build (ALL) + migrate + health + cleanup
ansible-playbook playbooks/deploy.yml --ask-vault-pass # full setup: writes .env from the vault, build, migrate
ansible-playbook playbooks/rollback.yml -e "commits=1" # git reset --hard HEAD~N + rebuild + health
```
> **Test status (2026-06):** ✅ `ansible` core 2.21 installed · server connectivity OK (`ansible calendar_server -m raw -a "echo"` rc=0) · `--syntax-check` OK on all 3 playbooks. ❌ A **real run** of quick-deploy/deploy/rollback has NOT been tested (they are disruptive — they rebuild everything / touch prod). The **direct SSH** commands above, on the other hand, have been used in production. The first time you run a playbook, watch it.
Others (one-off setup): `setup-nginx-ssl.yml`, `setup-authelia.yml`, `setup-health-check.yml`, `setup-backup.yml`, `sync-data.yml`.
- **Vault**: `deploy.yml` injects secrets from `inventory/group_vars/all/vault.yml` (gitignored, encrypted) → `--ask-vault-pass`. quick-deploy/rollback do **not** need the vault.
- `deploy.yml` runs **`git reset --hard origin/main`** → **discards the server's local modifications** (backup.sh/health-check.sh would be lost). quick-deploy runs `git pull` (which can conflict with them). That is why direct SSH with `--ff-only` tends to be safer today.

## Migrations (Prisma, calendar-core)
In production it is `migrate deploy` (NOT `dev`):
```bash
ssh root@45.90.123.190 "cd /opt/calendar && docker-compose -f docker-compose.prod.yml exec -T calendar-core npx prisma migrate deploy"
```
quick-deploy/deploy already run this. It only runs automatically when there is a pending migration.

## Health check (verify everything)
```bash
ssh -n root@45.90.123.190 'for u in 3334/ 3335/health 3336/api/v1/health 3337/health 3003/ 3001/ 3004/; do \
  echo -n "$u -> "; curl -s -o /dev/null -w "%{http_code}\n" http://127.0.0.1:$u; done; \
  docker ps --format "{{.Names}}: {{.Status}}" | grep -E "calendar|finances|health|agents|langfuse"'
```
(calendar-core has no `/health` route — it answers 404 but is healthy.) For data sanity, see the `financas` skill.

## Gotchas (project memory)
- **quick-deploy.yml fails** ("non-zero return code") on a long build (SSH drops) → prefer **direct SSH** with a detached build.
- **Unquoted colons in `raw:`** break the YAML → always quote: `raw: "cmd || echo 'msg'"`.
- The inventory is **`production.yml`** (not `hosts.yml`).
- `up -d` run from the wrong folder fails ("stat /root/docker-compose.prod.yml: no such file") → **always `cd /opt/calendar`** first (with `;`/`&&`, not inside a backgrounded subshell).
- `ssh -n` inside loops (otherwise it eats stdin). SSH noise: `-o LogLevel=ERROR` + `2>/dev/null`.
- `docker-compose restart` does NOT re-read the env → use `up -d --force-recreate`.
- Post-deploy cleanup: `docker image prune -f && docker builder prune -f` (the playbooks already do this).
