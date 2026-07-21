---
name: financas
description: Sistema financeiro (calendar-finances). Use SEMPRE que o usuário pedir para LANÇAR/registrar/efetuar uma transação, confirmar lançamentos PENDENTES ou ATRASADOS, pagar/confirmar fatura, reconciliar conta com o banco, conferir saldo, ou mexer em recorrências. Traz IDs de perfis/contas/categorias, auth e a regra de ouro (saldo = soma estrita das transações; nunca acertar saldo). Escrita SEMPRE via API, nunca SQL.
---

# Calendar Finances — API (produção)

Como criar/confirmar/consultar transações com segurança. **Para escrita, SEMPRE a API** (recalcula saldo, processa recorrência, atualiza fatura). SQL direto só para leitura ou rótulo de categoria — nunca para lançamento.

## ⚠️ REGRA DE OURO — saldo é resultado, nunca alvo
É um sistema financeiro: **o saldo deve representar ESTRITAMENTE a soma/subtração das transações.**
- **NUNCA "acertar" o saldo** diretamente (nem via API de recalc, nem SQL, nem editar `currentBalance`).
- Se o saldo do sistema **não bater** com o banco → **NÃO ajustar o saldo**. A diferença é sempre uma **transação faltando, sobrando ou errada**.
- **Procurar a correção nas transações, JUNTO com o usuário** — investigar qual lançamento falta/está errado, mostrar a diferença, e só lançar/corrigir a transação depois de confirmar com ele.
- Ex.: sistema 931,78 vs banco 933,74 → diferença +1,96 = rendimentos não lançados → lançar o rendimento (INCOME), não mexer no saldo.
- O recalc de saldo (`/bank-accounts/{id}/recalculate-balance`) só **recomputa a partir das transações** — use para corrigir saldo derivado, nunca para forçar um valor.

## Acesso / auth
- API **interna, sem token**, na VPS. Acesse via SSH:
  ```bash
  ssh -i ~/.ssh/id_rsa root@45.90.123.190 "curl -s http://127.0.0.1:3335/api/v1/<rota>"
  ```
- Base URL: `http://127.0.0.1:3335/api/v1`
- Filtre o ruído do SSH nos pipes: `| grep -vE "WARNING|post-quantum|decrypt later|openssh"`
- Respostas: `{"data": ...}`. POST/PUT mandam JSON com `-X POST -H 'Content-Type: application/json' -d '{...}'`.

## Perfis
| Perfil | ID | Tipo |
|---|---|---|
| Bruno Pessoal | `259866ac-6f74-4f7f-98b3-9a4896fb6758` | PERSONAL |
| WB Digital Solutions | `268b1f32-fe80-4293-9d03-87f1c63317b9` | BUSINESS |

## Contas — Bruno Pessoal
Conta padrão (checking) = **Mercado Pago** `6882e83a-b4fb-4cd5-b364-e0cfa61a90c3`.
| Conta | ID | Tipo |
|---|---|---|
| Mercado Pago | `6882e83a-b4fb-4cd5-b364-e0cfa61a90c3` | CHECKING |
| Nubank | `2e9875d5-49be-4b0f-b1ef-c9adb421f42a` | CHECKING |
| Wise (BRL) | `62b5726d-1efb-44df-8210-cba0a6a4d4b2` | CHECKING |
| Wise (EUR) | `cfa39f00-540b-4951-a511-76f4276b1761` | CHECKING (EUR) |
| Wise (USD) | `d8199fe6-efd5-4978-990b-f89a00f7fa8c` | CHECKING (USD) |
| Cartão Pessoal Nubank | `00e3c475-d21f-43f5-89ba-399d312c5095` | CREDIT_CARD |
| Cartão Mercado Pago | `73567965-1f0b-4d2e-868a-b1cfdc05fbbd` | CREDIT_CARD |
| Caixinha Mercado Pago | `b7c6aba6-df07-4a70-be56-2676541ada58` | INVESTMENT |
| Caixinha Nubank Pessoal | `c1f6172e-f3d9-4756-9f02-cb212f934fcd` | INVESTMENT |
| Binance (spot) | `cada06d0-d4cb-4a0e-b600-77bc589729a1` | EXCHANGE |
| (FIIs HGLG11/KNCR11/XPML11/KNRI11/SNAG11/CPTS11, CDBs, Clear, cripto Binance) | — | INVESTMENT |

## Contas — WB Digital Solutions
| Conta | ID | Tipo |
|---|---|---|
| Nubank Juridica | `22296070-8ef1-44e5-90b2-d7c92a87dba3` | CHECKING |
| Nubank Juridica Cartão | `174c6a9c-1b68-4a76-88d3-7cae3f1cf069` | CREDIT_CARD |
| Caixinha Nubank Juridica | `056a334a-bc51-4749-8668-34a14ca2adfc` | INVESTMENT |

## Categorias mais usadas — Bruno Pessoal
Streaming/IPTV `5441f3b1-7da0-4221-b260-b650efd6dba3` · Supermercado `e2f502a3-75a2-43f4-ba17-91105430783d` · Restaurante `fc5b2f6c-14b9-4a94-97b8-3b951dbd9244` · iFood `70a21cf2-1ea8-4bb9-93be-86a506adaae4` · Aluguel `2ed8988c-323c-437b-8ba3-90e0b4777866` · Contas `95b71fc6-e4c2-494d-8444-bfd020aed939` · Celular `cf992387-1dd2-474e-a69f-fe3681c7d802` · Uber `e6d7b8da-6c89-49e4-9b4c-f5b85c4fb236` · Combustível `97fb6741-1892-48dd-8a7b-934629fba215` · Saúde `614c3479-7122-44e3-b618-044cb33f971b` · Academia `6e7248c0-3558-4298-9f7c-1e7866ef217f` · Lazer `b9f169f5-cd51-4ff8-8352-18d8f37ff32d` · Rendimentos (INCOME) `ee4c3034-b0dc-4e18-872f-411c9dbbd65c` · Salário (INCOME) `5848835b-352e-4ffe-b7cc-d21b90af3f51` · Transferências (TRANSFER) `9e89e8d6-cba1-48bf-aee3-70f14473acb8`.

## Categorias mais usadas — WB
Receitas(INCOME) `8985a57a-e567-4016-b849-7fde3afe8f82` (Projetos/Servicos `520eb116-…`, Manutencao Mensal `c59f94be-…`, Aporte Socio `f8ac5b09-…`, Rendimentos Caixinha `da34a431-…`) · Despesas com Pessoal `6f08588f-…` (Pró-labore `63d95c6d-…`) · Infraestrutura `e6b897fe-…` (Software/SaaS `2ef99342-…`, Servidores `91dce52f-…`, Telefonia `216551dc-…`) · Marketing `377c50a5-…` · Taxas/Impostos `e37eae51-…` · Cursos/Capacitacao `51420c96-…` · Recrutamento/Freelancers `a4af3432-…` · Transporte `2996e140-…` (Uber `645d3a43-…`).

> **IDs completos podem mudar** — para a lista atual:
> `curl -s ".../categories?profileId=<id>"` e `.../bank-accounts?profileId=<id>`.

## Consultar
```bash
# pendentes/atrasados (PLANNED)
curl -s ".../transactions?profileId=<PID>&status=PLANNED&pageSize=100"
# transações de um período — ATENÇÃO: /transactions usa occurredFrom/occurredTo (NÃO from/to)
curl -s ".../transactions?profileId=<PID>&bankAccountId=<ACC>&occurredFrom=2026-06-01&occurredTo=2026-06-30&pageSize=200"
# (from/to só valem em /transactions/expense-analysis e /financial-summary)
# saldo das contas
curl -s ".../bank-accounts?profileId=<PID>"   # campo currentBalance
```
Status possíveis: **PLANNED · CONFIRMED · CANCELLED**. Tipos: **INCOME · EXPENSE · TRANSFER**.

## Confirmar um lançamento pendente/atrasado (o botão "Confirmar")
A maneira certa de "efetuar" uma recorrência atrasada que já existe como PLANNED:
```bash
ssh -i ~/.ssh/id_rsa root@45.90.123.190 "curl -s -X PUT \
  http://127.0.0.1:3335/api/v1/transactions/<TX_ID>/status \
  -H 'Content-Type: application/json' \
  -d '{\"status\":\"CONFIRMED\",\"occurredOn\":\"2026-06-23\"}'"
```
- `occurredOn` = data real do pagamento (não a data planejada). Recalcula o saldo da conta.

## Criar uma transação nova
```bash
ssh -i ~/.ssh/id_rsa root@45.90.123.190 "curl -s -X POST \
  http://127.0.0.1:3335/api/v1/transactions \
  -H 'Content-Type: application/json' \
  -d '{
    \"profileId\":\"259866ac-6f74-4f7f-98b3-9a4896fb6758\",
    \"bankAccountId\":\"6882e83a-b4fb-4cd5-b364-e0cfa61a90c3\",
    \"categoryId\":\"5441f3b1-7da0-4221-b260-b650efd6dba3\",
    \"type\":\"EXPENSE\",
    \"amount\":25.00,
    \"currency\":\"BRL\",
    \"description\":\"IPTV\",
    \"occurredOn\":\"2026-06-23\",
    \"status\":\"CONFIRMED\"
  }'"
```
Campos: `profileId, bankAccountId, categoryId, type, amount, currency(BRL), description, occurredOn` obrigatórios; `status` (default PLANNED → mande CONFIRMED para já efetivar); opcionais `notes, dueOn, reminderOn, destinationAccountId` (transfer), `installmentNumber/Total` (parcelas).

## Transferências (entre contas)
**Não há rota dedicada.** Transferência = `POST /transactions` com `type=TRANSFER` + `destinationAccountId`. O comportamento **muda conforme os perfis** (só efetiva saldo quando `status=CONFIRMED`):

**Mesmo perfil** → cria **1 transação** `TRANSFER`. Debita a origem, credita o destino. Se `categoryId` for enviada, tem que ser categoria tipo TRANSFER (ex: Transferências pessoal `9e89e8d6-...`); é opcional.
```bash
# resgate da caixinha p/ conta corrente, mesmo perfil
-d '{"profileId":"<PID>","bankAccountId":"<ORIGEM>","destinationAccountId":"<DESTINO>",
  "type":"TRANSFER","status":"CONFIRMED","amount":1170,"currency":"BRL",
  "description":"Dinheiro retirado","occurredOn":"2026-07-05"}'
```
⚠️ Se a **origem for cartão de crédito**, o saldo NÃO se move (guard de cartão) — só o destino é creditado.

**Cross-profile** (ex: MP do Bruno Pessoal → Nubank Jurídica da WB) → cria **2 transações ligadas** (`linkedTransactionId`):
- origem vira **EXPENSE** no perfil de origem (opcional `categoryId`, se enviada deve ser EXPENSE);
- destino vira **INCOME** no perfil de destino, usando **`destinationCategoryId` (OBRIGATÓRIO**, categoria INCOME do perfil de destino).
```bash
# aporte pessoal -> WB: EXPENSE Empréstimos (pessoal) + INCOME Aporte Sócio (WB)
-d '{"profileId":"259866ac-...","bankAccountId":"6882e83a-...(MP)",
  "destinationAccountId":"22296070-...(Nubank Jur)",
  "categoryId":"d359a66d-...(Empréstimos, EXPENSE pessoal)",
  "destinationCategoryId":"f8ac5b09-8647-4004-adf2-ba63dd5683e8 (Aporte Socio, INCOME WB)",
  "type":"TRANSFER","status":"CONFIRMED","amount":770,"currency":"BRL",
  "description":"Aporte WB Digital (via MP)","occurredOn":"2026-07-05"}'
```
O POST retorna só a transação de origem (com `linkedTransactionId` apontando pro INCOME do destino).

## Editar uma transação (PUT /transactions/{id})
`PUT /transactions/{id}` é **REPLACE COMPLETO** — campos omitidos são **zerados/apagados** (inclusive `categoryId`, `notes`). Pra mudar só o valor: pegue o objeto atual (GET) e **reenvie todos os campos setados** (`bankAccountId, categoryId, type, status, amount, currency, description, occurredOn`), mudando só o `amount`. Mantenha `bankAccountId` e `occurredOn` iguais pra não recomputar o vínculo de fatura. O total da fatura é derivado on-read (recalculado da soma das transações), então reflete sozinho no próximo GET.

## Recorrências
Modelo: a recorrência usa **RRULE** + datas, não `dayOfMonth`. Campos reais: `recurrenceRule` (ex: `FREQ=MONTHLY;BYMONTHDAY=13`), `startOn`, `nextOccurrence`, `amount`, `status` (ACTIVE/PAUSED/CANCELLED).
```bash
curl -s ".../recurring-transactions?profileId=<PID>"               # listar (objeto completo)
curl -s -X POST ".../recurring-transactions/process" -d '{...}'    # gerar os PLANNED do período
curl -s -X PATCH ".../recurring-transactions/<id>/status" -d '{"status":"PAUSED"}'
```
**Confirmar a recorrência do mês** = confirmar o PLANNED que ela gerou (`PUT /transactions/{id}/status`), NÃO mexer na recorrência.
**Mudar o dia (ex: dia 13 → 23)**: o `PUT /recurring-transactions/{id}` é um **replace completo** (`CreateRecurringTransactionInput`) — mande TODOS os campos, mudando só o que precisa. Pegue o objeto atual no GET e re-envie:
```bash
curl -s -X PUT ".../recurring-transactions/<id>" -H 'Content-Type: application/json' -d '{
  "profileId":"...","bankAccountId":"...","categoryId":"...","type":"EXPENSE","amount":25,
  "currency":"BRL","description":"IPTV","recurrenceRule":"FREQ=MONTHLY;BYMONTHDAY=23",
  "startOn":"2026-02-13","nextOccurrence":"2026-07-23","status":"ACTIVE","notes":"..."}'
```
Datas no input são `YYYY-MM-DD`. `nextOccurrence` = quando cai o próximo (ex: dia 23 do mês que vem).

## Gotchas (aprendidos na prática)
- `/transactions` lista limita por `pageSize` (manda alto, ex: 200) e **ignora `from/to`** — use `occurredFrom`/`occurredTo`.
- **Criar em lote** (ex: vários lançamentos): heredoc `ssh 'bash -s' <<EOF` falhou; use **loop local** com `ssh` por item. **CRÍTICO: `ssh -n`** dentro de `while read` (senão o ssh consome o stdin do heredoc e o loop para no 1º item). Padrão que funciona: `while IFS='|' read -r D A C DESC; do ssh -n ... "curl ... -d '{...\"amount\":$A,\"description\":\"$DESC\"...}'"; done <<ENTRIES` com linhas `data|valor|catId|descrição` (delimitador `|` aceita espaços/acentos na descrição). JSON: `\"` escapado, `-d '...'` single-quote, vars locais expandem.
- **Cartão de crédito**: lançar despesa = POST normal com `bankAccountId`=cartão; o sistema linka na fatura aberta sozinho. O lançamento "Pagamento fatura ..." que aparece no cartão é o crédito do pagamento (não é despesa).
- **IOF/câmbio internacional**: o valor gravado no sistema pode ficar 1 centavo abaixo do que o banco cobrou (arredondamento). Ao reconciliar cartão com a fatura, comparar **linha a linha** e corrigir cada IOF pro valor exato da fatura (via `PUT /transactions/{id}`). O "Total a pagar" do Nubank já embute arredondamento próprio (a soma bruta das linhas pode dar 1-2 centavos a mais que o total oficial) — não dá pra casar ao centavo por soma; pague o valor real cobrado.
- **Python para parsear** a resposta: rode LOCAL (depois do `| python3 -c '...'`), com aspas duplas normais (NÃO escapar); o que vai dentro do `ssh "..."` é só o curl.
- SSH ruidoso: `-o LogLevel=ERROR` + `2>/dev/null` no curl deixa só o JSON.
- Rendimentos do MP (conta corrente) são lançados como INCOME diários "rendimento da conta", categoria Rendimentos `ee4c3034-...`. Fim de semana às vezes não tem.

## Regras (memória do projeto)
- **Escrita só via API** (recalcula saldo/fatura). SQL direto: só leitura ou rótulo de categoria.
- **Confirmar a categoria com o usuário** antes de lançar algo novo.
- **Nunca ajustar lançamento só para "bater" saldo** — investigar lançamento faltante.
- Pagamento de fatura de cartão: **`POST /invoices/{id}/pay`** (é POST, não PUT) já cria o débito (não lançar manual). Body: `{"paidAmount":769.51,"paidAt":"2026-07-05"}` (`paidAt` = `YYYY-MM-DD`). Debita da **conta linkada ao cartão** (`linkedAccountId`), não precisa passar conta. Confira antes: `GET /bank-accounts/{cardId}` → `linkedAccountId`. Ache a fatura em `GET /invoices?bankAccountId=<cardId>` (a a pagar é a CLOSED com `dueDate` do mês). `paidAmount` = valor real cobrado pelo banco.
- Reconciliação: comparar com o extrato do banco; ao reconciliar, contas do Bruno em BRL ficam no Mercado Pago/Nubank.
