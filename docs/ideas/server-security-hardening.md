# Server Security Hardening — Roadmap

## Status Atual (Fev 2026)

### O que ja temos
- **fail2ban** ativo com 4 jails (sshd, nginx-404, nginx-badbots, nginx-limit-req)
- **UFW** configurado (apenas 22/80/443)
- **Portas Docker** corrigidas para 127.0.0.1 (nao expostas a internet)
- **Authelia** protegendo frontends e Langfuse
- **SSL** com Let's Encrypt + renovacao automatica
- **Health check diario** com envio via WhatsApp (meio-dia BRT) — inclui verificacao de seguranca (fail2ban, tentativas SSH, portas abertas, logins recentes)
- **Containers** com limites de memoria
- **SSH com senha desabilitado** — `PasswordAuthentication no` (ja vinha configurado pela Contabo)
- **Root login apenas por chave** — `PermitRootLogin prohibit-password` (ja vinha configurado)
- **Backup automatizado para Google Drive** — cron diario as 4h BRT, retencao 7 dias, notifica WhatsApp em erro
  - Bancos: calendar_db (public + finance schemas) + langfuse_db
  - Configs: .env, nginx, authelia
  - Script: `scripts/backup.sh`
  - Playbook: `deploy/ansible/playbooks/setup-backup.yml`
  - Destino: `gdrive:backups/calendar/`

### O que falta

## 1. Hardening SSH

### 1.1 Desabilitar login com senha
~~**Risco: ALTO**~~ — **FEITO** (ja estava configurado pela Contabo)

```
# /etc/ssh/sshd_config (estado atual)
PasswordAuthentication no
PubkeyAuthentication yes  (comentado mas e o default)
PermitRootLogin prohibit-password
```

### 1.2 Criar usuario separado (desabilitar root direto)
**Risco: MEDIO** — Se a chave SSH vazar, atacante tem root imediato.

```bash
# Criar user separado
adduser bruno
usermod -aG sudo bruno
mkdir -p /home/bruno/.ssh
cp ~/.ssh/authorized_keys /home/bruno/.ssh/
chown -R bruno:bruno /home/bruno/.ssh

# /etc/ssh/sshd_config
PermitRootLogin no
```

Depois disso, login com `ssh bruno@server` e `sudo` quando necessario.

### 1.3 Mudar porta SSH (opcional)
**Risco: BAIXO** — Reduz 99% dos bots automatizados que so tentam porta 22.

```bash
# /etc/ssh/sshd_config
Port 2222  # ou outra porta alta
```

Atualizar UFW: `ufw allow 2222/tcp && ufw delete allow 22/tcp`

### 1.4 2FA no SSH (opcional)
Google Authenticator via `libpam-google-authenticator`. Exige codigo TOTP alem da chave SSH.

---

## 2. Monitoramento de Intrusao

### 2.1 Alerta imediato para eventos criticos
**Risco: ALTO** — Se alguem invadir as 3h, so sabemos ao meio-dia.

Opcao A: **Cron a cada 5 minutos** (leve) que verifica apenas:
- Login SSH novo (IP desconhecido)
- Container parado
- Processo com CPU > 80% desconhecido

So envia WhatsApp se detectar algo. Sem spam se tudo OK.

Opcao B: **auditd + regras** para alertar em tempo real via syslog.

### 2.2 Verificacao de IPs conhecidos
Manter uma lista de IPs seus (casa, escritorio, celular) e alertar quando um login vier de IP desconhecido.

```bash
# /opt/calendar/config/known-ips.txt
179.233.116.0/24   # Casa Bruno
```

### 2.3 Verificacao de integridade de arquivos
**Risco: MEDIO** — Backdoor pode ser instalado em binarios do sistema.

```bash
# rkhunter (ja esta instalado no servidor!)
rkhunter --check --skip-keypress
```

Adicionar ao health check: rodar `rkhunter` semanalmente e reportar warnings.

### 2.4 Processos suspeitos
Detectar mineradores de crypto ou processos desconhecidos consumindo CPU.

```bash
# Top 5 processos por CPU, excluindo conhecidos
ps aux --sort=-%cpu | head -10
```

Alertar se algum processo desconhecido usar > 50% CPU por mais de 5 minutos.

---

## 3. Protecao de Rede

### 3.1 Rate limiting nos APIs publicos
**Risco: BAIXO** — APIs sem throttling podem ser abusados.

Adicionar ao Nginx para os endpoints de API:
```nginx
limit_req_zone $binary_remote_addr zone=api:10m rate=30r/m;

location /api/ {
    limit_req zone=api burst=10 nodelay;
    ...
}
```

### 3.2 Bloquear paises (opcional)
Se o servidor so atende Brasil, bloquear IPs de paises com mais bots (China, Russia, etc.) via GeoIP no Nginx ou UFW.

### 3.3 Docker network isolation
Garantir que containers de projetos diferentes nao se comunicam entre si. Cada projeto deve ter sua propria rede Docker isolada.

---

## 4. Backup e Recovery

### 4.1 Verificar backups existem
~~**Risco: ALTO**~~ — **FEITO**

Situacao atual dos backups:
| Projeto | Banco | Backup | Destino |
|---------|-------|--------|---------|
| Calendar | calendar_db (public + finance) | `scripts/backup.sh` diario 4h BRT | Google Drive |
| Langfuse | langfuse_db | `scripts/backup.sh` diario 4h BRT | Google Drive |
| CRM | crm_db | `/opt/backups/backup-crm.sh` | Google Drive |
| n8n | n8n_db | `/root/n8n/backup-n8n.sh` | Google Drive |
| Evolution | evolution_db | **SEM BACKUP** | - |

Tambem faz backup de: `.env`, configs Nginx, configs Authelia.

### 4.2 Backup automatizado
~~**Risco: ALTO**~~ — **FEITO**

Cron diario as 4h BRT (07:00 UTC):
1. Dump PostgreSQL (calendar_db + langfuse_db)
2. Copia .env, nginx configs, authelia configs
3. Comprime em `.tar.gz`
4. Envia para `gdrive:backups/calendar/`
5. Retem ultimos 7 dias (local + drive)
6. Notifica WhatsApp em caso de erro

### 4.3 Testar restore
Backup que nao foi testado nao e backup. Fazer restore de teste mensal.

### 4.4 Backup do Evolution (pendente)
O banco `evolution_db` nao tem backup. Avaliar se e necessario (dados podem ser recriados?) ou adicionar ao script.

---

## 5. Melhorias no Health Check

### 5.1 Adicionar ao script existente
- [x] Verificar fail2ban (IPs bloqueados por jail)
- [x] Verificar tentativas SSH falhas
- [x] Verificar portas abertas (alem de 22/80/443)
- [x] Mostrar logins recentes (IPs)
- [ ] Comparar IPs de login com lista de IPs conhecidos
- [ ] Top processos por CPU (detectar mineradores)
- [ ] Verificar se rkhunter tem warnings recentes
- [ ] Verificar idade do ultimo backup

### 5.2 Criar script de alerta rapido (5 min)
Script separado e leve que roda a cada 5 minutos:
- Login SSH de IP desconhecido -> WhatsApp imediato
- Container critico parado -> WhatsApp imediato
- CPU > 90% por 5 min -> WhatsApp imediato

Nao substitui o check diario — complementa com alertas urgentes.

### 5.3 Melhorar texto do fail2ban
Atual: `fail2ban: 22 IPs bloqueados (nginx-badbots=1 sshd=21)`
Melhor: `Seguranca: 21 tentativas de invasao SSH bloqueadas, 1 bot malicioso bloqueado`

---

## Prioridade de Implementacao

| # | Item | Risco | Esforco | Status |
|---|------|-------|---------|--------|
| 1 | Desabilitar SSH com senha | Alto | 5 min | **FEITO** (ja estava) |
| 2 | Backup automatizado | Alto | 1h | **FEITO** |
| 3 | Alerta imediato (5 min cron) | Alto | 30 min | PENDENTE |
| 4 | Criar usuario separado (desabilitar root) | Medio | 15 min | PENDENTE |
| 5 | Verificacao de integridade (rkhunter) | Medio | 15 min | PENDENTE |
| 6 | Deteccao de processos suspeitos | Medio | 15 min | PENDENTE |
| 7 | IPs conhecidos vs desconhecidos | Medio | 20 min | PENDENTE |
| 8 | Rate limiting APIs | Baixo | 20 min | PENDENTE |
| 9 | Mudar porta SSH | Baixo | 10 min | PENDENTE |
| 10 | 2FA SSH | Baixo | 30 min | PENDENTE |
| 11 | Bloquear paises | Baixo | 30 min | PENDENTE |

---

## Notas
- O servidor ja tem **rkhunter** instalado (encontrado durante setup)
- fail2ban com 21 bans SSH e **normal** para servidor publico — bots tentam 24/7
- Docker bypassa UFW por padrao — sempre usar `127.0.0.1:` nos ports
- IPs nos "Logins recentes" devem ser verificados manualmente por enquanto
- **Evolution API** (evolution_db) e o unico banco sem backup automatizado
