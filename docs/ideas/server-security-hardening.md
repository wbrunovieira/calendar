# Server Security Hardening — Roadmap

## Status Atual (Fev 2026)

### O que já temos
- **fail2ban** ativo com 4 jails (sshd, nginx-404, nginx-badbots, nginx-limit-req)
- **UFW** configurado (apenas 22/80/443)
- **Portas Docker** corrigidas para 127.0.0.1 (não expostas à internet)
- **Authelia** protegendo frontends e Langfuse
- **SSL** com Let's Encrypt + renovação automática
- **Health check diário** com envio via WhatsApp (meio-dia BRT)
- **Containers** com limites de memória

### O que falta

## 1. Hardening SSH

### 1.1 Desabilitar login com senha
**Risco: ALTO** — Bots tentam senhas 24/7, fail2ban ajuda mas não elimina o risco.

```bash
# /etc/ssh/sshd_config
PasswordAuthentication no
PubkeyAuthentication yes
```

Verificar antes que a chave SSH funciona, senão perde acesso ao servidor.

### 1.2 Desabilitar login root direto
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

Depois disso, login com `ssh bruno@server` e `sudo` quando necessário.

### 1.3 Mudar porta SSH (opcional)
**Risco: BAIXO** — Reduz 99% dos bots automatizados que só tentam porta 22.

```bash
# /etc/ssh/sshd_config
Port 2222  # ou outra porta alta
```

Atualizar UFW: `ufw allow 2222/tcp && ufw delete allow 22/tcp`

### 1.4 2FA no SSH (opcional)
Google Authenticator via `libpam-google-authenticator`. Exige código TOTP além da chave SSH.

---

## 2. Monitoramento de Intrusão

### 2.1 Alerta imediato para eventos críticos
**Risco: ALTO** — Se alguém invadir às 3h, só sabemos ao meio-dia.

Opção A: **Cron a cada 5 minutos** (leve) que verifica apenas:
- Login SSH novo (IP desconhecido)
- Container parado
- Processo com CPU > 80% desconhecido

Só envia WhatsApp se detectar algo. Sem spam se tudo OK.

Opção B: **auditd + regras** para alertar em tempo real via syslog.

### 2.2 Verificação de IPs conhecidos
Manter uma lista de IPs seus (casa, escritório, celular) e alertar quando um login vier de IP desconhecido.

```bash
# /opt/calendar/config/known-ips.txt
179.233.116.0/24   # Casa Bruno
```

### 2.3 Verificação de integridade de arquivos
**Risco: MEDIO** — Backdoor pode ser instalado em binários do sistema.

```bash
# rkhunter (já está instalado no servidor!)
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

## 3. Proteção de Rede

### 3.1 Rate limiting nos APIs públicos
**Risco: BAIXO** — APIs sem throttling podem ser abusados.

Adicionar ao Nginx para os endpoints de API:
```nginx
limit_req_zone $binary_remote_addr zone=api:10m rate=30r/m;

location /api/ {
    limit_req zone=api burst=10 nodelay;
    ...
}
```

### 3.2 Bloquear países (opcional)
Se o servidor só atende Brasil, bloquear IPs de países com mais bots (China, Rússia, etc.) via GeoIP no Nginx ou UFW.

### 3.3 Docker network isolation
Garantir que containers de projetos diferentes não se comunicam entre si. Cada projeto deve ter sua própria rede Docker isolada.

---

## 4. Backup e Recovery

### 4.1 Verificar backups existem
**Risco: ALTO** — Se o servidor morrer, perdemos tudo?

Verificar:
- Banco Calendar PostgreSQL — tem backup automático?
- Banco Langfuse — tem backup?
- Volumes Docker — salvos em algum lugar?
- Configs (/etc/nginx, .env files) — versionados?

### 4.2 Backup automatizado
Cron diário que:
1. Dump PostgreSQL (calendar + langfuse)
2. Comprime
3. Envia para storage externo (S3, outro VPS, Google Drive)
4. Retém últimos 7 dias

### 4.3 Testar restore
Backup que não foi testado não é backup. Fazer restore de teste mensal.

---

## 5. Melhorias no Health Check

### 5.1 Adicionar ao script existente
- [ ] Verificar `PasswordAuthentication` no sshd_config
- [ ] Verificar `PermitRootLogin` no sshd_config
- [ ] Comparar IPs de login com lista de IPs conhecidos
- [ ] Top processos por CPU (detectar mineradores)
- [ ] Verificar se rkhunter tem warnings recentes
- [ ] Verificar idade do último backup

### 5.2 Criar script de alerta rápido (5 min)
Script separado e leve que roda a cada 5 minutos:
- Login SSH de IP desconhecido → WhatsApp imediato
- Container crítico parado → WhatsApp imediato
- CPU > 90% por 5 min → WhatsApp imediato

Não substitui o check diário — complementa com alertas urgentes.

### 5.3 Melhorar texto do fail2ban
Atual: `fail2ban: 22 IPs bloqueados (nginx-badbots=1 sshd=21)`
Melhor: `Seguranca: 21 tentativas de invasao SSH bloqueadas, 1 bot malicioso bloqueado`

---

## Prioridade de Implementação

| # | Item | Risco | Esforço | Prioridade |
|---|------|-------|---------|------------|
| 1 | Desabilitar SSH com senha | Alto | 5 min | URGENTE |
| 2 | Backup automatizado | Alto | 1h | URGENTE |
| 3 | Alerta imediato (5 min cron) | Alto | 30 min | ALTA |
| 4 | Desabilitar root login | Medio | 15 min | ALTA |
| 5 | Verificação de integridade (rkhunter) | Medio | 15 min | MEDIA |
| 6 | Detecção de processos suspeitos | Medio | 15 min | MEDIA |
| 7 | IPs conhecidos vs desconhecidos | Medio | 20 min | MEDIA |
| 8 | Rate limiting APIs | Baixo | 20 min | BAIXA |
| 9 | Mudar porta SSH | Baixo | 10 min | BAIXA |
| 10 | 2FA SSH | Baixo | 30 min | BAIXA |
| 11 | Bloquear países | Baixo | 30 min | BAIXA |

---

## Notas
- O servidor já tem **rkhunter** instalado (encontrado durante setup)
- fail2ban com 21 bans SSH é **normal** para servidor público — bots tentam 24/7
- Docker bypassa UFW por padrão — sempre usar `127.0.0.1:` nos ports
- IPs nos "Logins recentes" devem ser verificados manualmente por enquanto
