# Plano: Transaction Agent — Lançamentos via WhatsApp

Agente LangGraph que recebe mensagens de texto (via n8n ← Evolution API ← WhatsApp) e cria transações financeiras no calendar-finances.

---

## 1. Credenciais — `.env`

Todas as credenciais ficam no `.env` (gitignored). O `.env.example` contém apenas placeholders.

### Variáveis de ambiente

| Variável | Descrição | Sensível |
|----------|-----------|----------|
| `EVOLUTION_API_URL` | URL base da Evolution API | Não |
| `EVOLUTION_API_KEY` | Chave de autenticação (header `apikey`) | **Sim** |
| `EVOLUTION_INSTANCE` | Nome da instância WhatsApp | Não |
| `N8N_WEBHOOK_URL` | URL do webhook que recebe mensagens do WhatsApp | Não |
| `N8N_WEBHOOK_SECRET` | Secret para validar requests do webhook (header `x-webhook-secret`) | **Sim** |
| `N8N_API_URL` | URL base da API REST do n8n | Não |
| `N8N_API_KEY` | Chave de autenticação da API do n8n | **Sim** |
| `WHATSAPP_GROUP_JID` | ID do grupo "Financeiro" para filtro no n8n | Não |

### `.env.example`
```env
# Evolution API (WhatsApp)
EVOLUTION_API_URL=
EVOLUTION_API_KEY=
EVOLUTION_INSTANCE=

# n8n
N8N_WEBHOOK_URL=
N8N_WEBHOOK_SECRET=
N8N_API_URL=
N8N_API_KEY=

# WhatsApp — Grupo dedicado para lançamentos
WHATSAPP_GROUP_JID=120363424817003115@g.us
```

### `docker-compose.yml` — service `calendar-agents`
```yaml
- EVOLUTION_API_URL=${EVOLUTION_API_URL:-}
- EVOLUTION_API_KEY=${EVOLUTION_API_KEY:-}
- EVOLUTION_INSTANCE=${EVOLUTION_INSTANCE:-}
- N8N_WEBHOOK_URL=${N8N_WEBHOOK_URL:-}
- N8N_WEBHOOK_SECRET=${N8N_WEBHOOK_SECRET:-}
- N8N_API_URL=${N8N_API_URL:-}
- N8N_API_KEY=${N8N_API_KEY:-}
```

---

## 1.1. Evolution API — Rotas utilizadas

A Evolution API é o gateway WhatsApp. O calendar-agents **não** chama diretamente — o n8n faz o routing. Documentação aqui para referência do fluxo n8n.

### Autenticação
Header obrigatório em todas as rotas:
```
apikey: <EVOLUTION_API_KEY>
```

### Enviar texto (n8n → Evolution)
```
POST /message/sendText/{instance}
```
```json
{
  "number": "5511999999999",
  "text": "Sua mensagem aqui"
}
```

### Enviar mídia/documento (futuro)
```
POST /message/sendMedia/{instance}
```
```json
{
  "number": "5511999999999",
  "mediatype": "image|document",
  "mimetype": "image/png|application/pdf",
  "caption": "Legenda",
  "fileName": "arquivo.pdf",
  "media": "https://url-do-arquivo.com/arquivo.ext"
}
```

### Enviar áudio (futuro)
```
POST /message/sendWhatsAppAudio/{instance}
```
```json
{
  "number": "5511999999999",
  "audio": "https://url-do-audio.com/audio.mp3"
}
```

### Verificar status da conexão
```
GET /instance/connectionState/{instance}
```

### Verificar se número tem WhatsApp
```
POST /chat/whatsappNumbers/{instance}
```
```json
{"numbers": ["5511999999999"]}
```

---

## 1.2. n8n Webhook — Payload recebido

A Evolution envia POST automaticamente para o webhook configurado no n8n.

### Headers de autenticação
```
x-webhook-secret: <N8N_WEBHOOK_SECRET>
```

### Payload (mensagem de texto recebida)
```json
{
  "event": "MESSAGES_UPSERT",
  "instance": "wbdigital",
  "data": {
    "key": {
      "remoteJid": "5511999999999@s.whatsapp.net",
      "fromMe": false,
      "id": "ABC123"
    },
    "pushName": "Nome do Contato",
    "message": {
      "conversation": "Texto da mensagem"
    },
    "messageTimestamp": 1770294411
  }
}
```

### Campos importantes

| Campo | Descrição |
|-------|-----------|
| `event` | Tipo: `MESSAGES_UPSERT`, `CONNECTION_UPDATE`, etc. |
| `data.key.remoteJid` | Número do remetente (`5511999999999@s.whatsapp.net`) |
| `data.key.fromMe` | `true` = enviada por nós, `false` = recebida |
| `data.pushName` | Nome do contato no WhatsApp |
| `data.message.conversation` | Texto da mensagem |
| `data.key.id` | ID único da mensagem (para `externalId` futuro) |

### Isolamento — Grupo "Financeiro"

O WhatsApp é de uso pessoal. Para evitar que mensagens do dia a dia ativem o agent, usamos um **grupo dedicado**.

| Info | Valor |
|------|-------|
| Nome | Financeiro |
| Group JID | Configurado em `WHATSAPP_GROUP_JID` no `.env` |
| Membros | Você (admin) |

**Regra de filtragem:** apenas mensagens onde `remoteJid` termina em `@g.us` **e** é igual ao `WHATSAPP_GROUP_JID` são encaminhadas ao agent. Mensagens pessoais (`@s.whatsapp.net`) são 100% ignoradas.

### Fluxo no n8n
1. Webhook recebe payload da Evolution
2. Filtra: `event == "MESSAGES_UPSERT"` e `fromMe == false`
3. **Filtra grupo:** `data.key.remoteJid == WHATSAPP_GROUP_JID` (descarta tudo que não for do grupo Financeiro)
4. Extrai `phone` (do `participant` no payload de grupo) e `text` (`message.conversation`)
5. POST para `http://calendar-agents:3337/agents/transaction` com `{ phone, text, profile_id }`
6. Recebe resposta com `reply`
7. Envia `reply` no grupo via `POST /message/sendText/{instance}` com `number: WHATSAPP_GROUP_JID`

---

## 2. Fluxo end-to-end

```
Grupo "Financeiro" (WhatsApp)
    → Evolution API (webhook automático)
    → n8n (filtra: grupo == WHATSAPP_GROUP_JID, fromMe == false)
        → POST http://calendar-agents:3337/agents/transaction
          Body: { "phone": "5511999999999", "text": "almoco 32 nubank" }
        ← Response: { "reply": "✓ Almoço R$ 32,00 (Nubank → Alimentação)" }
        → n8n envia reply no grupo via Evolution sendText API
```

- Mensagens pessoais (`@s.whatsapp.net`) são **ignoradas** pelo filtro no n8n
- O calendar-agents **não** fala direto com Evolution/WhatsApp — n8n faz o routing
- Respostas do agent são enviadas de volta **no grupo**, não em conversa privada

---

## 3. Input/Output do endpoint

### Request (n8n → calendar-agents)

```
POST /agents/transaction
Content-Type: application/json
```

```json
{
  "phone": "5511999999999",
  "text": "almoco 32 nubank",
  "profile_id": "uuid-do-perfil"
}
```

`profile_id` pode ser resolvido no n8n (binding phone → profile) ou no agent (lookup por phone). Fase 1: vem fixo do n8n.

### Response (calendar-agents → n8n)

**Sucesso:**
```json
{
  "status": "created",
  "reply": "✓ Almoço R$ 32,00 (Nubank → Alimentação)",
  "transaction": {
    "id": "uuid",
    "type": "EXPENSE",
    "amount": 32.00,
    "description": "Almoço",
    "bank_account": "Nubank",
    "category": "Alimentação",
    "occurred_on": "2026-02-05",
    "status": "CONFIRMED"
  }
}
```

**Erro de parsing (não entendeu):**
```json
{
  "status": "error",
  "reply": "Não entendi. Tente: 'descrição valor conta'. Ex: 'almoco 32 nubank'",
  "error": "parse_failed"
}
```

**Erro de validação (conta não encontrada, etc.):**
```json
{
  "status": "error",
  "reply": "Conta 'bradesco' não encontrada. Contas disponíveis: Nubank, Itaú",
  "error": "account_not_found"
}
```

---

## 4. LangGraph — Transaction Graph

### State

```python
class TransactionState(TypedDict):
    # Input
    phone: str
    raw_text: str
    profile_id: str

    # Context (carregado do backend)
    accounts: list[dict]       # GET /bank-accounts?profileId=...
    categories: list[dict]     # GET /categories?profileId=...

    # Parsed by LLM
    parsed: dict | None        # {description, amount, type, account_name, category_name}

    # Resolved IDs
    bank_account_id: str | None
    category_id: str | None

    # Result
    transaction: dict | None
    reply: str
    error: str | None
```

### Nodes

```
START
  → load_context      (busca accounts + categories do backend via httpx)
  → parse_message     (LLM: extrai campos estruturados do texto)
  → resolve_entities  (match account_name → ID, category_name → ID)
  → create_transaction (POST /api/v1/transactions no calendar-finances)
  → format_reply      (monta mensagem de resposta)
END
```

### Edges condicionais

```
parse_message:
  - se parsed == None → format_reply (erro: não entendeu)
  - se parsed ok → resolve_entities

resolve_entities:
  - se account não encontrada → format_reply (erro: lista contas)
  - se ok → create_transaction

create_transaction:
  - se erro da API → format_reply (erro: mensagem da API)
  - se ok → format_reply (sucesso)
```

---

## 5. Prompt do LLM (node parse_message)

### System prompt

```
Você é um parser de lançamentos financeiros. Receba uma mensagem curta de WhatsApp e extraia os campos.

Regras:
- O formato típico é: "descrição valor [conta] [categoria]"
- O valor pode vir com ou sem "R$", vírgula ou ponto: "32", "32.50", "32,50", "R$ 32"
- A conta é opcional — se não informada, retorne null
- A categoria é opcional — se não informada, retorne null
- O tipo padrão é EXPENSE. Só use INCOME se o texto indicar claramente (ex: "salario", "recebi", "freelance")
- A data padrão é hoje. Só mude se o texto indicar (ex: "ontem", "dia 3")
- Se não conseguir extrair pelo menos descrição + valor, retorne null

Contas disponíveis do usuário:
{accounts_list}

Categorias disponíveis:
{categories_list}
```

### User prompt

```
Mensagem: "{raw_text}"
```

### Output format (structured output / tool call)

```json
{
  "description": "Almoço",
  "amount": 32.00,
  "type": "EXPENSE",
  "account_name": "Nubank",
  "category_name": "Alimentação",
  "occurred_on": "2026-02-05",
  "status": "CONFIRMED"
}
```

Ou `null` se não conseguir parsear.

### Exemplos no prompt (few-shot)

| Mensagem | Output |
|----------|--------|
| "almoco 32 nubank" | `{description: "Almoço", amount: 32, type: "EXPENSE", account_name: "Nubank", category_name: "Alimentação"}` |
| "uber 18.50" | `{description: "Uber", amount: 18.50, type: "EXPENSE", account_name: null, category_name: "Transporte"}` |
| "salario 8000 itau" | `{description: "Salário", amount: 8000, type: "INCOME", account_name: "Itaú", category_name: "Renda"}` |
| "mercado 230 nubank alimentacao" | `{description: "Mercado", amount: 230, type: "EXPENSE", account_name: "Nubank", category_name: "Alimentação"}` |
| "cafe" | `null` (sem valor) |

---

## 6. Langfuse — Rastreamento

Cada execução do graph gera um **trace** no Langfuse:

```python
from langfuse import Langfuse
from langfuse.callback import CallbackHandler

langfuse = Langfuse()

# Cada request = 1 trace
trace = langfuse.trace(
    name="transaction-agent",
    input={"phone": phone, "text": raw_text},
    metadata={"profile_id": profile_id},
)

# Node parse_message = 1 generation (LLM call)
generation = trace.generation(
    name="parse_message",
    model="gpt-4o-mini",
    input=[system_prompt, user_prompt],
    output=parsed_result,
    metadata={"prompt_version": "v1"},
)

# Node create_transaction = 1 span
span = trace.span(
    name="create_transaction",
    input=transaction_payload,
    output=api_response,
)

# Final
trace.update(output={"reply": reply, "status": status})
```

### O que rastreamos

| Evento | Tipo Langfuse | Dados |
|--------|--------------|-------|
| Request recebido | trace.input | phone, text, profile_id |
| Contexto carregado | span | accounts count, categories count |
| LLM parse | generation | prompt, parsed output, model, tokens, latência |
| Entity resolution | span | account_name → ID, category_name → ID |
| API call finances | span | request body, response status, transaction ID |
| Reply final | trace.output | reply text, status |

### Prompt versioning
- Prompt armazenado no Langfuse como **prompt management** (nome: `transaction-parser`, version: `v1`)
- Permite A/B test de prompts sem deploy
- `langfuse.get_prompt("transaction-parser", version=1)`

### Scores (feedback loop futuro)
- Quando usuário responde "ok" ou "corrigir" no WhatsApp → score no trace
- `trace.score(name="user_feedback", value=1)` (confirmou) ou `value=0` (corrigiu)

---

## 7. Campos para POST /api/v1/transactions

### Campos que o agente preenche

| Campo | Fonte | Obrigatório |
|-------|-------|-------------|
| `profileId` | Vem do request (fixo fase 1) | Sim |
| `bankAccountId` | Resolvido por nome → ID | Sim |
| `type` | LLM infere (default: EXPENSE) | Sim |
| `amount` | LLM extrai do texto | Sim |
| `currency` | Fixo: "BRL" | Sim |
| `description` | LLM extrai do texto | Sim |
| `occurredOn` | LLM extrai ou hoje | Sim |
| `status` | CONFIRMED (default para WhatsApp) | Sim |
| `categoryId` | LLM sugere → resolve por nome → ID | Não |

### Campos ignorados (fase 1)
- `destinationAccountId` (transfers — futuro)
- `notes`, `costCenter`, `tags`, `splits` (complexidade desnecessária)
- `dueOn`, `reminderOn`, `recurrenceRule` (futuro)
- `installmentNumber`, `installmentTotal` (futuro)
- `externalId` (futuro: message ID do WhatsApp)

---

## 8. Arquivos a criar/modificar

### Criar

```
services/calendar-agents/app/
├── agents/
│   ├── __init__.py
│   └── transaction/
│       ├── __init__.py
│       ├── graph.py          # LangGraph: TransactionState + nodes + edges
│       ├── nodes.py          # load_context, parse_message, resolve_entities, create_transaction, format_reply
│       ├── prompts.py        # System/user prompts para o LLM parser
│       └── router.py         # POST /agents/transaction
├── clients/
│   ├── __init__.py
│   └── finances.py           # httpx client para calendar-finances API
└── config.py                 # Settings (URLs dos backends, keys)
```

### Modificar

| Arquivo | Mudança |
|---------|---------|
| `services/calendar-agents/app/main.py` | Importar e incluir `transaction_router` |
| `services/calendar-agents/requirements.txt` | Adicionar `langchain-openai` |
| `services/calendar-agents/.env.example` | Adicionar variáveis Evolution/n8n |
| `docker-compose.yml` | Adicionar env vars Evolution/n8n no service calendar-agents |
| `.env` | Adicionar credenciais reais |
| `.env.example` (raiz) | Adicionar placeholders Evolution/n8n |

---

## 9. Dependência de requirements.txt

Adicionar:
```
langchain-openai>=0.2,<1
```

Já existentes e necessários:
- `langgraph` — orquestração do grafo
- `langchain-core` — base para LLM calls
- `langfuse` — tracing
- `httpx` — chamadas HTTP para calendar-finances
- `fastapi` — endpoint
- `python-dotenv` — env vars

---

## 10. Verificação

```bash
# 1. Rebuild calendar-agents
docker compose up -d --build calendar-agents

# 2. Health check
curl http://localhost:3337/health

# 3. Teste direto (sem WhatsApp/n8n)
curl -X POST http://localhost:3337/agents/transaction \
  -H "Content-Type: application/json" \
  -d '{
    "phone": "5511999999999",
    "text": "almoco 32 nubank",
    "profile_id": "PROFILE_ID_AQUI"
  }'

# 4. Verificar trace no Langfuse
open http://localhost:3100

# 5. Verificar transação criada no calendar-finances
curl "http://localhost:3335/api/v1/transactions?profileId=PROFILE_ID_AQUI"

# 6. Teste via WhatsApp (end-to-end)
# Enviar "almoco 32 nubank" no WhatsApp → verificar resposta
```
