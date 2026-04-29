# Plano: Monitor de Saldo/Gasto de APIs LLM com Alerta WhatsApp

## Contexto

Provedores LLM ativos em produção:
- **Anthropic** (principal) — pay-as-you-go, cartão pós-pago. Sem API de saldo. Risco: cartão vencido, limite de gasto atingido, spike de custo.
- **DeepSeek** (finanças) — crédito pré-pago em USD. API de saldo disponível. Risco: crédito zerado silenciosamente.

Problema atual: um saldo zerado ou cartão rejeitado quebra todas as pesquisas de lead e análises de ligação sem nenhum alerta.

---

## Objetivos

1. Alertar preventivamente quando DeepSeek estiver abaixo de threshold
2. Alertar quando gasto diário Anthropic ultrapassar threshold (spike ou cartão problemático)
3. Detectar falha de autenticação em ambos os provedores (cartão vencido, key inválida)
4. Entrega via WhatsApp (Evolution API, mesmo canal do health-check existente)

---

## Abordagem: Script bash diário (mesmo padrão do health-check.sh)

Seguir o padrão já estabelecido em `scripts/health-check.sh`:
- Cron diário no servidor de produção
- Script bash lê variáveis do `.env`
- Envia WhatsApp via Evolution API em caso de alerta
- Log em `/var/log/llm-balance-check.log`

### Por que bash e não agente Python?

- Zero dependência nova — httpx/curl já disponível
- Independente do serviço `agents` (monitora mesmo se o container cair)
- Mesmo padrão dos outros monitores do projeto
- Ansible playbook já existente para configurar crons

---

## Implementação

### 1. DeepSeek — saldo pré-pago

**Endpoint:** `GET https://api.deepseek.com/user/balance`
**Auth:** `Authorization: Bearer $DEEPSEEK_API_KEY`

**Resposta esperada:**
```json
{
  "balance_infos": [{
    "currency": "USD",
    "total_balance": "4.23",
    "granted_balance": "0.00",
    "topped_up_balance": "4.23"
  }],
  "is_available": true
}
```

**Lógica:**
- `is_available = false` → alerta crítico (API indisponível para uso)
- `total_balance < DEEPSEEK_ALERT_THRESHOLD` (default: $5.00) → alerta de saldo baixo
- Falha HTTP / auth error → alerta de key inválida

### 2. Anthropic — gasto diário

**Endpoint:** `GET https://api.anthropic.com/v1/usage`
**Auth:** `x-api-key: $ANTHROPIC_API_KEY`, `anthropic-version: 2023-06-01`

**Parâmetros:** `?start_time=<hoje 00:00 UTC>&end_time=<agora>`

**Lógica:**
- Soma custo do dia (`input_tokens * price + output_tokens * price`)
- `custo_dia > ANTHROPIC_DAILY_ALERT_THRESHOLD` (default: $10.00) → alerta de spike
- Falha HTTP 401/403 → alerta de key inválida ou cartão rejeitado
- Preços atuais: Haiku $0.80/$4.00 por MTok, Sonnet $3.00/$15.00 por MTok

### 3. Teste de liveness (ambos)

Fazer uma chamada mínima (1 token) em cada provedor.
- Erro 401 → key inválida
- Erro 402/429 com mensagem de billing → cartão/crédito problema
- Timeout → provedor com instabilidade

---

## Estrutura do script

```
scripts/
  llm-balance-check.sh       ← script principal
deploy/ansible/playbooks/
  setup-llm-balance-check.yml ← cron + logrotate via Ansible
```

### Variáveis de ambiente necessárias (já no .env de produção)

```bash
DEEPSEEK_API_KEY=...
DEEPSEEK_ALERT_THRESHOLD=5.00     # USD — alertar abaixo deste valor

ANTHROPIC_API_KEY=...
ANTHROPIC_DAILY_ALERT_THRESHOLD=10.00  # USD/dia — alertar acima deste valor
```

### Thresholds recomendados para ajustar conforme uso

| Variável | Default | Justificativa |
|----------|---------|---------------|
| `DEEPSEEK_ALERT_THRESHOLD` | $5.00 | ~500k tokens de folga para recarregar |
| `ANTHROPIC_DAILY_ALERT_THRESHOLD` | $10.00 | Uso normal estimado < $3/dia; $10 = spike real |

### Formato da mensagem WhatsApp

```
🔴 ALERTA LLM — DeepSeek
Saldo: $2.31 (abaixo do limite de $5.00)
Recarregue em: platform.deepseek.com
Data: 2026-04-29 08:00 BRT

🟡 ALERTA LLM — Anthropic
Gasto hoje: $12.40 (limite: $10.00)
Verifique uso anormal no dashboard
Data: 2026-04-29 08:00 BRT
```

---

## Cron

```cron
# Verificação de saldo LLM — 8h BRT (11h UTC)
0 11 * * * root /opt/calendar/scripts/llm-balance-check.sh >> /var/log/llm-balance-check.log 2>&1
```

Horário: 8h BRT dá tempo hábil para recarregar DeepSeek antes do fim do expediente.

---

## Playbook Ansible

Seguir o padrão de `setup-health-check.yml`:
- Copia o script para `/opt/calendar/scripts/`
- Cria o cron em `/etc/cron.d/llm-balance-check`
- Configura logrotate (7 dias de retenção)
- Testa execução inicial

---

## Ordem de implementação

1. `scripts/llm-balance-check.sh` — lógica principal
2. Adicionar variáveis de threshold no `.env.example`
3. `deploy/ansible/playbooks/setup-llm-balance-check.yml`
4. Rodar playbook em produção
5. Verificar primeiro log após execução

---

## Referências

- DeepSeek Balance API: https://platform.deepseek.com/docs#get-user-balance
- Anthropic Usage API: https://docs.anthropic.com/en/api/usage
- Evolution API WhatsApp: padrão já em uso no `scripts/health-check.sh`
- Cron e logrotate: padrão já em uso no `deploy/ansible/playbooks/setup-health-check.yml`
