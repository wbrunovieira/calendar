# agents — Roadmap de Agentes AI

> Documento de ideias para agentes ativos no `agents` (Python/FastAPI/LangGraph).
> Cada agente consome dados dos backends existentes (calendar-core, calendar-finances, calendar-health)
> e usa LLM + LangGraph para gerar insights, automações e ações proativas.
> Todas as execuções são rastreadas via Langfuse.

---

## 1. Daily Briefing Agent (Morning Digest)

**Trigger:** Cron diário (ex: 7h) ou `GET /briefing/today`

Consulta os 3 backends e monta um resumo personalizado do dia:

### Dados consumidos

| Backend | Endpoints | Dados |
|---------|-----------|-------|
| calendar-core | `GET /events?startDate&endDate&eventType` | Eventos, hábitos, TODOs, reminders do dia |
| calendar-core | `GET /events/habits/stats` | Streak atual, taxa de completude |
| calendar-core | `GET /events/habits/weekly-progress` | Progresso semanal de hábitos flexíveis |
| calendar-finances | `GET /recurring-transactions?profileId` | Contas a pagar hoje (next_occurrence = hoje) |
| calendar-finances | `GET /invoices/current?bankAccountId` | Faturas abertas com vencimento próximo |
| calendar-finances | `GET /bank-accounts?profileId` | Saldos das contas |
| calendar-health | `GET /workouts?startDate&endDate` | Treino planejado para hoje |
| calendar-health | `GET /activities?startDate&endDate` | Atividades planejadas |

### Output esperado

```json
{
  "date": "2026-02-05",
  "calendar": {
    "events": 3,
    "habits_pending": 4,
    "todos_due_today": 1,
    "todos_overdue": 2,
    "current_streak": 12,
    "reminders": ["Renovar CNH"]
  },
  "finances": {
    "bills_due_today": [{"description": "Internet", "amount": 120.00}],
    "credit_card_due_soon": [{"card": "Nubank", "amount": 2340.50, "due": "2026-02-10"}],
    "safe_to_spend": 85.30,
    "total_balance": 4520.00
  },
  "health": {
    "workout_planned": "Upper Body - 6:00",
    "activities_planned": ["Wim Hof Breathing"]
  },
  "ai_message": "Bom dia! Você tem 4 hábitos pendentes e está no 12o dia de streak. A fatura do Nubank vence em 5 dias — saldo suficiente na conta vinculada."
}
```

### Complexidade: Baixa
Somente leitura. Sem escrita em nenhum backend. LLM gera apenas a mensagem contextualizada.

---

## 2. Natural Language Creator Agent

**Trigger:** `POST /agents/parse`

Recebe texto livre do usuário e cria entidades no backend correto.

### Exemplos de input → output

| Input | Backend | Ação |
|-------|---------|------|
| "Reunião com João amanhã às 14h" | calendar-core | `POST /events` (EVENT) |
| "Paguei 45 reais de almoço no Nubank" | calendar-finances | `POST /transactions` (EXPENSE, CONFIRMED) |
| "Correr 3x por semana" | calendar-core | `POST /events` (HABIT, FLEXIBLE, weeklyTargetCount=3) |
| "Academia todo dia útil às 6h" | calendar-core | `POST /events` (HABIT, FIXED, RRULE BYDAY=MO,TU,WE,TH,FR) |
| "Lembrar de comprar presente dia 15" | calendar-core | `POST /events` (REMINDER, dueDate=15) |
| "Spotify 21.90 todo mês dia 5" | calendar-finances | `POST /recurring-transactions` (EXPENSE, MONTHLY, BYMONTHDAY=5) |

### LangGraph Flow

```
[User Input]
    → parse_intent (LLM: identifica domínio + tipo)
    → extract_entities (LLM: extrai campos estruturados)
    → validate (verifica campos obrigatórios, resolve datas relativas)
    → confirm (opcional: retorna preview para o usuário)
    → execute (POST no backend correto via httpx)
    → respond (mensagem de confirmação)
```

### Complexidade: Média
Escrita nos backends. Requer parsing de datas relativas ("amanhã", "próxima segunda"), resolução de categorias existentes e contas bancárias por nome.

---

## 3. Habit Coach Agent

**Trigger:** Cron diário (pós almoço) ou `GET /agents/habit-coach`

Analisa padrões de completude e gera coaching proativo.

### Funcionalidades

#### 3.1 Detecção de risco de quebra de streak
- Consulta `GET /events/habits/stats` para cada hábito
- Se taxa de completude caiu nos últimos 3 dias, gera alerta
- Mensagem: "Seu streak de meditação (18 dias) está em risco — você não completou ontem. Que tal fazer agora?"

#### 3.2 Redistribuição de hábitos flexíveis
- Consulta `GET /events/habits/weekly-progress`
- Se é quinta-feira e meta de 3x/semana tem apenas 1 completude, sugere os dias restantes
- Usa `weeklyPreferredDays` para priorizar
- Mensagem: "Você precisa de mais 2 sessões de exercício até domingo. Sexta e sábado são seus dias preferidos."

#### 3.3 Padrões de horário
- Analisa `completedAt` dos `EventCompletion` via `GET /events/:id/executions`
- Identifica horários de pico de completude por tipo de hábito
- Sugere agendamento: "Seus hábitos de estudo são 80% mais completados entre 9h-11h."

#### 3.4 Reconhecimento de conquistas
- Detecta marcos: 7, 14, 30, 60, 90, 180, 365 dias de streak
- Detecta recordes pessoais: "Maior streak de todos os tempos!"
- Gera mensagem motivacional contextualizada via LLM

### Complexidade: Média
Leitura + análise temporal. LLM gera mensagens de coaching. Sem escrita nos backends.

---

## 4. Financial Guardian Agent

**Trigger:** Cron diário ou `GET /agents/financial-guardian`

Monitora finanças e gera alertas inteligentes.

### Funcionalidades

#### 4.1 Detecção de anomalias de gastos
- Consulta `GET /transactions?profileId&type=EXPENSE&occurredFrom&occurredTo` dos últimos 3 meses
- Agrupa por categoria e calcula média mensal
- Se gasto do mês atual > 130% da média, alerta
- Mensagem: "Alimentação está R$ 340 acima da média (R$ 850 vs média de R$ 510). Faltam 12 dias no mês."

#### 4.2 Projeção de estouro de budget
- Consulta `GET /budgets/summary?profileId&period=YYYY-MM`
- Calcula pace: % do mês decorrido vs % do budget gasto
- Se pace > 1.2 (gastando 20% mais rápido que o esperado), alerta com projeção
- Mensagem: "No ritmo atual, Transporte vai estourar em R$ 180. Sugiro limitar a R$ 15/dia nos próximos 10 dias."

#### 4.3 Recurring transactions esquecidas
- Consulta recurring transactions ACTIVE
- Cruza com eventos do calendário (há uso correlato?)
- Se uma assinatura não tem correlação com nenhuma atividade, sugere revisão
- Mensagem: "Você paga Gym (R$ 99/mês) mas não registrou treino nos últimos 30 dias. Manter?"

#### 4.4 Alerta de cartão de crédito
- Consulta `GET /invoices/current?bankAccountId` para cada cartão
- Consulta `GET /bank-accounts` para saldo da conta vinculada
- Se fatura > 80% do limite, ou saldo da conta vinculada < valor da fatura, alerta
- Mensagem: "Fatura Nubank: R$ 3.200 (vence dia 10). Saldo na conta vinculada: R$ 2.800 — faltam R$ 400."

### Complexidade: Média
Leitura + análise estatística. Sem escrita. LLM gera mensagens de alerta.

---

## 5. Smart Categorization Agent

**Trigger:** Hook no `POST /transactions` ou `POST /agents/categorize`

Classifica automaticamente transações baseado em descrição.

### Flow

```
[Nova transação sem categoria]
    → busca categorias do perfil (GET /categories?profileId)
    → busca últimas 50 transações categorizadas (histórico)
    → LLM: dado descrição + categorias disponíveis + histórico, sugere categoria
    → retorna sugestão com confidence score
```

### Exemplos

| Descrição | Sugestão | Confidence |
|-----------|----------|------------|
| "Uber" | Transporte | 0.95 |
| "iFood Burger King" | Alimentação > Delivery | 0.92 |
| "Spotify Premium" | Assinaturas > Streaming | 0.98 |
| "Pix João" | (sem sugestão) | 0.20 |

### Feedback loop via Langfuse
- Cada sugestão é um trace no Langfuse
- Quando o usuário confirma ou corrige, registra score
- LLM melhora com contexto das correções

### Complexidade: Baixa
LLM + histórico do usuário. Pode ser síncrono no fluxo de criação.

---

## 6. Weekly Review Agent

**Trigger:** Cron semanal (domingo 20h ou segunda 7h) ou `GET /agents/weekly-review`

Gera relatório semanal consolidado dos 3 domínios.

### Estrutura do relatório

```
## Semana 03/02 — 09/02/2026

### Hábitos
- Completude geral: 78% (semana passada: 85%) ↓
- Melhor hábito: Meditação (7/7)
- Pior hábito: Leitura (2/5)
- Streak mais longo ativo: Exercício (23 dias)

### Finanças
- Total gasto: R$ 1.240 (semana passada: R$ 980) ↑
- Maior gasto: Mercado (R$ 420)
- Budget status: 2 no limite, 1 estourado (Delivery)
- Economia vs. receita: 32%

### Saúde
- Treinos realizados: 4/5
- PRs batidos: Supino 80kg (novo recorde!)
- Atividades: 3x Wim Hof, 2x Meditação

### Sugestões da semana
1. Leitura caiu para 2/5 — tente associar a um horário fixo (seu pico é 21h-22h)
2. Delivery estourou o budget em R$ 60 — nos dias sem treino, gastos com delivery são 2x maiores
3. Parabéns pelo PR no supino! Considere aumentar a meta de peso para o próximo ciclo.
```

### Complexidade: Média
Agregação de dados de 3 backends + LLM para gerar narrativa e sugestões.

---

## 7. Cross-Domain Correlation Agent

**Trigger:** `GET /agents/insights` ou cron mensal

Cruza dados entre calendar, finances e health para descobrir correlações.

### Correlações possíveis

| Correlação | Dados cruzados | Insight exemplo |
|------------|---------------|-----------------|
| Exercício vs. Delivery | health workouts + finance transactions (Alimentação > Delivery) | "Semanas com 4+ treinos têm 30% menos gastos com delivery" |
| Wim Hof vs. Produtividade | health activities (BREATHING) + calendar completions | "Dias com Wim Hof de manhã têm 25% mais TODOs completados" |
| Estresse financeiro vs. Hábitos | finance budget overspend + calendar habit completion rate | "Quando budget estoura, completude de hábitos cai 20% na semana seguinte" |
| Sono implícito vs. Treino | calendar first event time + health workout rating | "Treinos com rating 4-5 coincidem com dias em que primeiro evento foi após 8h" |
| Gastos de fim de semana vs. Streak | finance weekend spending + calendar weekend completions | "Fins de semana com streak mantido têm gasto médio 15% menor" |

### Abordagem técnica
- Coleta dados de 3-6 meses
- Calcula correlações estatísticas simples (Pearson/Spearman via numpy)
- LLM interpreta as correlações significativas e gera narrativa
- Filtra apenas insights com p-value < 0.05

### Complexidade: Alta
Análise estatística cruzada + coleta de dados de 3 backends + interpretação via LLM.

---

## Priorização

| Prioridade | Agent | Valor | Complexidade | Dependências |
|-----------|-------|-------|-------------|--------------|
| 1 | Daily Briefing | Alto | Baixa | Nenhuma — somente leitura |
| 2 | Natural Language Creator | Alto | Média | Nenhuma |
| 3 | Habit Coach | Alto | Média | Dados de completude existentes |
| 4 | Financial Guardian | Alto | Média | Dados de transações existentes |
| 5 | Smart Categorization | Médio | Baixa | Categorias do perfil |
| 6 | Weekly Review | Médio | Média | Agents 3 e 4 parcialmente |
| 7 | Cross-Domain Correlation | Alto | Alta | Dados históricos de 3+ meses |

---

## Stack técnica compartilhada

Todos os agentes usam a mesma infraestrutura:

- **LangGraph**: Orquestração de passos (parse → validate → execute → respond)
- **httpx**: Chamadas HTTP para os backends (calendar-core:3334, calendar-finances:3335, calendar-health:3336)
- **Langfuse**: Trace de cada execução (input, output, latência, tokens, scores de feedback)
- **FastAPI**: Endpoints REST para trigger manual ou consumo pelo frontend
- **APScheduler** ou **Celery**: Cron para triggers automáticos (daily briefing, weekly review)

### Padrão de endpoint

```
GET  /agents/{agent-name}           # Execução manual
GET  /briefing/today                # Daily briefing
POST /agents/parse                  # Natural language creator
GET  /agents/habit-coach            # Habit coach
GET  /agents/financial-guardian     # Financial guardian
POST /agents/categorize             # Smart categorization
GET  /agents/weekly-review          # Weekly review
GET  /agents/insights               # Cross-domain insights
```

---
---

# Ideias Expandidas — Segunda Rodada

> Explorações mais profundas, criativas e ambiciosas.

---

## 8. Time Architect Agent (Otimização de Agenda)

**Trigger:** `GET /agents/time-architect` ou cron diário (noite anterior)

Analisa a agenda do dia seguinte e sugere reorganizações inteligentes.

### O problema
O usuário tem eventos, hábitos, TODOs com due date, treinos planejados e contas a pagar — tudo espalhado. Ninguém monta mentalmente o "dia perfeito" considerando tudo junto.

### O que faz

#### 8.1 Detecção de conflitos silenciosos
- Hábito "Leitura" marcado para 21h mas há evento "Jantar com amigos" das 20h às 23h
- TODO com due date hoje mas agenda completamente cheia — quando fazer?
- Treino planejado às 6h mas ontem dormiu tarde (inferido: último evento terminou às 23h30)

#### 8.2 Sugestão de blocos de tempo
Baseado nos dados históricos de completude:
- "Seus TODOs de trabalho rendem melhor entre 9h-11h (82% completude vs 45% à tarde)"
- "Hábitos de saúde completados 90% das vezes quando feitos antes das 8h"
- Sugere slots vagos ideais para cada tipo de atividade

#### 8.3 Buffer inteligente
- Detecta dias com agenda lotada (>6 eventos) e sugere remover/adiar itens de baixa prioridade
- "Você tem 8 itens amanhã. Nos dias com 8+ itens, sua completude cai para 52%. Sugiro adiar Leitura para quarta."

#### 8.4 Energy mapping
Cruza com dados de health:
- Dia pós-treino pesado (legs day) → sugere menos TODOs cognitivos
- Dia com Wim Hof de manhã → aproveitar o pico de energia para tarefas difíceis
- Sem treino há 3 dias → sugere atividade leve para manter ritmo

### Output

```json
{
  "date": "2026-02-06",
  "suggested_schedule": [
    {"time": "06:00", "item": "Wim Hof Breathing", "reason": "Seu pico de produtividade acontece após esta atividade"},
    {"time": "07:00", "item": "Treino Upper Body", "source": "health"},
    {"time": "09:00-11:00", "item": "Deep work: TODO 'Refatorar API auth'", "reason": "Melhor janela cognitiva"},
    {"time": "12:00", "item": "Pagar fatura Nubank (vence amanhã)", "source": "finances"},
    {"time": "14:00", "item": "Reunião com João", "source": "calendar"},
    {"time": "21:00", "item": "Leitura (30min)", "reason": "Horário habitual, 78% completude neste slot"}
  ],
  "warnings": [
    "Amanhã tem 7 itens. Considere adiar 'Organizar documentos' (prioridade baixa, sem due date)."
  ]
}
```

### Complexidade: Alta
Requer modelo de "energy level" do usuário construído ao longo do tempo.

---

## 9. Accountability Partner Agent (Conversa Ativa)

**Trigger:** `POST /agents/chat` (conversacional)

Um agente que lembra quem você é, conhece seu histórico e conversa como um parceiro de accountability.

### O problema
Dashboards são passivos. O usuário abre, olha, fecha. Um parceiro de accountability real pergunta, provoca, celebra.

### Funcionalidades

#### 9.1 Check-in proativo
Não espera o usuário perguntar. O agente gera mensagens baseadas em contexto:
- **15h, nenhum hábito completado**: "Ei, nenhum hábito feito hoje ainda. Dia corrido? O mínimo que você pode fazer em 5 min é a meditação."
- **Sexta-feira, meta semanal incompleta**: "Faltam 2 sessões de exercício para bater sua meta semanal. Sábado tá livre às 8h."
- **Após completar treino pesado**: "Leg day done! Último PR de agachamento foi 100kg há 3 semanas. Progresso?"

#### 9.2 Memória de contexto
Armazenado no Langfuse como metadata de sessão:
- Sabe que o usuário está tentando economizar para uma viagem
- Sabe que está em fase de bulk no treino
- Sabe que tem deadline de projeto na próxima semana
- Usa isso para contextualizar mensagens

#### 9.3 Estilo adaptativo
- Quando streak está alto: celebração e reforço positivo
- Quando está caindo: direto, prático, sem julgamento
- Quando volta após ausência: acolhimento + plano de retomada

### Diferencial
Não é um chatbot genérico. É um agente que tem acesso real aos seus dados e pode dizer: "Você disse que ia cortar delivery, mas esta semana já gastou R$ 180 — mais que nas últimas 3 semanas."

### Complexidade: Alta
Requer memória persistente, context window grande, e design cuidadoso de personalidade.

---

## 10. Financial Forecast Agent (Projeção de Cenários)

**Trigger:** `GET /agents/forecast?months=3` ou `POST /agents/forecast/scenario`

Projeta o futuro financeiro baseado em dados reais.

### Funcionalidades

#### 10.1 Projeção automática de fluxo de caixa
- Coleta: saldo atual de todas as contas
- Coleta: recurring transactions ACTIVE (receitas e despesas fixas)
- Coleta: média de gastos variáveis por categoria (últimos 3 meses)
- Projeta saldo para os próximos N meses

```
Fev/2026:  Saldo atual R$ 4.520
           + Salário R$ 8.000
           - Fixos R$ 3.200 (aluguel, internet, streaming, gym...)
           - Variáveis ~R$ 2.800 (média: alimentação, transporte, lazer)
           - Fatura cartão ~R$ 1.500
           = Projeção: R$ 5.020

Mar/2026:  = Projeção: R$ 5.520
Abr/2026:  = Projeção: R$ 6.020
```

#### 10.2 Simulação de cenários (what-if)
O usuário pergunta via chat ou endpoint:
- "Se eu cancelar a academia, quanto economizo em 6 meses?" → calcula R$ 99 × 6 = R$ 594, mas também mostra impacto no health tracking
- "Se eu aumentar investimento mensal para R$ 1.000?" → recalcula projeção com rendimento
- "Consigo juntar R$ 10.000 até julho?" → calcula o gap e sugere cortes

#### 10.3 Detecção de meses críticos
- Identifica meses com saldo projetado negativo
- "Em abril você tem IPVA (R$ 2.400) + seguro (R$ 1.800). Saldo projetado: -R$ 680. Sugiro reservar R$ 230/mês a partir de agora."
- Cruza com calendar events (férias, viagens, datas comemorativas) que tipicamente geram gastos extras

#### 10.4 Meta de reserva de emergência
- Calcula custo mensal fixo total
- Monitora progresso para X meses de reserva
- "Sua reserva de emergência cobre 2.3 meses. Meta recomendada: 6 meses (R$ 19.200). Faltam R$ 12.300."

### Complexidade: Média
Matemática direta com recurring transactions + médias históricas. LLM para narrativa e cenários.

---

## 11. Habit Stacking Agent (Formação de Rotinas)

**Trigger:** `GET /agents/habit-stacking` ou quando novo hábito é criado

Baseado na ciência de habit stacking (James Clear / Atomic Habits).

### O que faz

#### 11.1 Detecta cadeias naturais
Analisa horários de completude e descobre sequências que já acontecem:
- "Você faz Wim Hof → Treino → Meditação em 73% dos dias. Isso é uma rotina matinal natural."
- "Leitura sempre acontece após último evento do dia. Está ancorada no fim da agenda."

#### 11.2 Sugere ancoragem para novos hábitos
Quando o usuário cria um novo hábito:
- Analisa quais hábitos existentes têm alta completude
- Sugere: "Ancore 'Journaling' após 'Meditação' — sua meditação tem 92% de completude, é a âncora ideal."

#### 11.3 Detecta rotinas frágeis
- Se um hábito âncora falha e os subsequentes também falham, alerta
- "Quando você pula o treino, meditação e leitura também falham 80% das vezes. O treino é keystone habit."

#### 11.4 Rotina mínima viável
Para dias em que tudo está falhando:
- "Dia difícil? Sua rotina mínima: 5min meditação + 10min caminhada. Protege o streak e mantém o momentum."
- Baseado nos hábitos com menor tempo necessário

### Complexidade: Média
Análise temporal de sequências + LLM para recomendações baseadas em behavioral science.

---

## 12. Expense Negotiator Agent (Redução de Custos)

**Trigger:** `GET /agents/expense-audit` ou cron mensal

Analisa recurring transactions e sugere renegociações.

### Funcionalidades

#### 12.1 Benchmark de assinaturas
- Lista todas as recurring transactions
- Compara com preços típicos (LLM knowledge)
- "Seu plano de internet é R$ 150. Planos similares custam R$ 99-120."

#### 12.2 Detecção de duplicatas
- "Você paga Spotify (R$ 21.90) e YouTube Music (R$ 20.90). Ambos são streaming de música."
- "HBO Max e Disney+ — você usou HBO 12x no mês e Disney+ 0x."

#### 12.3 Sazonalidade de gastos
- Analisa 12 meses de histórico por categoria
- "Gastos com transporte sobem 40% em dezembro (festas). Reserve R$ 200 extra em novembro."
- "Conta de luz sobe R$ 80 no verão. Período: dez-mar."

#### 12.4 ROI de assinaturas
- Cruza com calendar/health: gym pago + treinos registrados = custo por treino
- "Academia: R$ 99/mês ÷ 18 treinos = R$ 5.50/treino. Bom ROI."
- "Coursera: R$ 79/mês ÷ 0 acessos = sugiro pausar."

### Complexidade: Baixa-Média
Recurring transactions + LLM analysis. Sem escrita.

---

## 13. Sleep & Recovery Inference Agent

**Trigger:** Parte do Daily Briefing ou `GET /agents/recovery-status`

Não tem dados diretos de sono, mas pode inferir padrões.

### Inferências possíveis

#### 13.1 Horário de sono inferido
- Último evento/hábito completado do dia = proxy de "hora de dormir"
- Primeiro evento/hábito completado do dia seguinte = proxy de "hora de acordar"
- Constrói padrão de sono estimado ao longo das semanas

#### 13.2 Impacto no desempenho
- "Dias em que seu primeiro hábito é completado antes das 7h: completude geral 85%"
- "Dias em que primeiro hábito é após 9h: completude geral 62%"
- "Após noites com último evento após 23h: rating de treino cai de 4.2 para 3.1"

#### 13.3 Debt detection
- "Nos últimos 5 dias, seu horário de acordar atrasou 30min/dia. Possível sleep debt acumulado."
- "Sugestão: durma 1h mais cedo hoje para resetar."

### Complexidade: Média
Análise de timestamps de completude. Nenhum dado novo necessário — usa os que já existem.

---

## 14. Goal Decomposition Agent

**Trigger:** `POST /agents/goal`

O usuário define um objetivo de alto nível e o agente decompõe em ações concretas.

### Exemplos

**Input:** "Quero economizar R$ 10.000 até julho"
**Output:**
- Projeção: faltam 5 meses, precisa de R$ 2.000/mês
- Gasto atual médio: R$ 5.800/mês. Receita: R$ 8.000
- Sobra atual: R$ 2.200 → meta viável com margem de R$ 200
- Cria budget targets automáticos (com confirmação)
- Identifica categorias cortáveis: "Delivery -R$ 300 e Lazer -R$ 200 liberam R$ 500 de margem"

**Input:** "Quero correr uma meia maratona em 6 meses"
**Output:**
- Cria hábito FLEXIBLE: "Corrida 3x/semana" (confirma com usuário)
- Sugere progressão: semana 1-4 = 5km, semana 5-8 = 8km, etc.
- Sugere atividades complementares: "Adicionar stretching pós-corrida"
- Monitora progresso via weekly review

**Input:** "Quero ler 24 livros este ano"
**Output:**
- 24 livros ÷ 12 meses = 2/mês = ~1 a cada 15 dias
- Cria hábito "Leitura 30min" FIXED, daily
- Cria TODO recorrente "Escolher próximo livro" a cada 15 dias
- Tracking via completude do hábito

### LangGraph Flow

```
[Goal input]
    → classify_domain (finances? health? productivity? mixed?)
    → analyze_current_state (consulta backends relevantes)
    → decompose (LLM: quebra em sub-metas mensais/semanais)
    → generate_actions (cria hábitos, budgets, TODOs)
    → present_plan (retorna para confirmação)
    → execute (cria entidades nos backends com confirmação)
```

### Complexidade: Alta
Multi-domínio, escrita em backends, requer planejamento temporal.

---

## 15. Anomaly & Life Event Detector

**Trigger:** Cron diário (background analysis)

Detecta mudanças significativas nos padrões de vida do usuário.

### Detecções

#### 15.1 Mudanças de rotina
- "Você parou de treinar há 8 dias. Isso é incomum (média: treina 4.5x/semana). Tudo bem?"
- "Seus gastos com farmácia subiram 300% esta semana."
- "Nenhum evento de trabalho nos últimos 3 dias (era 5-6/semana). Férias?"

#### 15.2 Transições de fase
Detecta quando padrões mudam de forma sustentada (não pontual):
- "Nas últimas 3 semanas, seus gastos com delivery dobraram e treinos caíram 50%. Período de estresse?"
- "Sua completude de hábitos está em tendência de alta: 65% → 72% → 78% → 83%. Excelente progresso!"

#### 15.3 Alertas de saúde implícitos
Combina sinais fracos de múltiplos domínios:
- Menos treinos + mais gastos com farmácia + menos hábitos completados = possível período de doença
- Rating de treino caindo gradualmente = possível overtraining
- Completude de todos os hábitos caindo = possível burnout

### Ação
Não apenas detecta — sugere:
- "Parece um período difícil. Sugiro ativar 'rotina mínima viável' (agent 11) para proteger os streaks mais importantes."
- "Possível overtraining detectado. Sugiro 1 semana de deload (metade da carga)."

### Complexidade: Alta
Detecção de anomalias estatísticas + inferência contextual via LLM.

---

## 16. Import & Reconciliation Agent

**Trigger:** `POST /agents/import`

Automatiza importação de dados financeiros de fontes externas.

### 16.1 Parse de extrato bancário (CSV/OFX)
O sistema já suporta Nubank CSV/OFX. O agente:
- Recebe arquivo
- Parseia transações
- Auto-categoriza cada uma (usa Smart Categorization Agent internamente)
- Detecta duplicatas com transações existentes (por data + valor + descrição similar)
- Apresenta preview para confirmação
- Cria transações em batch

### 16.2 Reconciliação
- Compara transações importadas com recurring transactions
- Marca recurrings como "confirmadas" quando match no extrato
- Identifica transações no extrato que não têm recurring correspondente
- "3 novas assinaturas detectadas no extrato: Netflix R$ 55.90, iCloud R$ 3.50, ChatGPT R$ 104. Criar recurring?"

### 16.3 Detecção de cobranças indevidas
- "Cobrança duplicada detectada: Uber R$ 32.50 aparece 2x no dia 03/02"
- "Spotify cobrou R$ 34.90 mas seu plano é R$ 21.90. Verificar?"

### Complexidade: Média
Parse de arquivos + matching fuzzy + LLM para categorização.

---

## 17. Gamification & Scoring Agent

**Trigger:** Calculado em background, exposto via `GET /agents/score`

Transforma dados em um sistema de pontuação e gamificação.

### Life Score (0-100)

Calculado diariamente com pesos:

| Dimensão | Peso | Fonte | Métrica |
|----------|------|-------|---------|
| Hábitos | 30% | calendar-core | % completude da semana |
| Finanças | 25% | calendar-finances | % do budget respeitado + savings rate |
| Saúde | 25% | calendar-health | Treinos/semana vs meta + atividades |
| Produtividade | 20% | calendar-core | TODOs completados vs criados |

### Badges & Achievements

| Badge | Condição |
|-------|----------|
| Ironman | 30 dias de streak em todos os hábitos |
| Frugal Master | 3 meses consecutivos abaixo do budget |
| Beast Mode | 5 treinos/semana por 4 semanas |
| Zero Inbox | Todos os TODOs completados por 7 dias |
| Millionaire Mindset | Savings rate > 30% por 3 meses |
| Early Bird | Primeiro hábito completado antes das 7h por 14 dias |
| Consistency King | Life Score > 80 por 30 dias |

### Leaderboard pessoal
- Score semanal ao longo do tempo (gráfico de evolução)
- "Melhor semana" e "pior semana" do mês
- Tendência: melhorando, estável, decaindo

### Complexidade: Média
Cálculos matemáticos + definição de regras. Sem LLM necessário (ou mínimo para mensagens).

---

## 18. Conversational Report Agent (Perguntas em Linguagem Natural)

**Trigger:** `POST /agents/ask`

O usuário faz perguntas sobre seus próprios dados e recebe respostas.

### Exemplos

| Pergunta | Agente consulta | Resposta |
|----------|----------------|----------|
| "Quanto gastei com Uber este mês?" | finances: transactions filtered | "R$ 234.50 em 8 viagens. Média de R$ 29.30/viagem." |
| "Qual meu hábito com maior streak?" | calendar: habits/stats | "Meditação com 45 dias consecutivos." |
| "Quanto sobra se eu cancelar Netflix e Spotify?" | finances: recurring-transactions | "R$ 77.80/mês = R$ 933.60/ano" |
| "Em que dia da semana gasto mais?" | finances: transactions aggregated | "Sábado: média R$ 180. Segunda: média R$ 45." |
| "Quantos treinos fiz em janeiro?" | health: workouts filtered | "18 treinos. 4.5/semana. Rating médio: 4.1." |
| "Estou melhor ou pior que mês passado?" | todos os 3 backends | Comparação completa com highlights |

### LangGraph Flow

```
[Pergunta]
    → classify_question (qual domínio? qual tipo de query?)
    → generate_queries (quais endpoints chamar? com quais filtros?)
    → execute_queries (httpx parallel requests)
    → analyze_results (LLM: interpreta dados no contexto da pergunta)
    → format_response (resposta concisa + dados de suporte)
```

### Complexidade: Média-Alta
Requer text-to-query mapping + interpretação de resultados. Muito flexível.

---

## 19. Proactive Notification Agent (Push Intelligence)

**Trigger:** Cron a cada 2-4 horas

Ao contrário dos outros agentes que esperam ser chamados, este empurra notificações quando detecta algo relevante.

### Regras de notificação

| Hora | Trigger | Mensagem |
|------|---------|----------|
| 7h | Agenda cheia + contas a pagar | Daily Briefing condensado |
| 10h | Nenhum hábito completado ainda | "Manhã voando — que tal começar com [hábito mais rápido]?" |
| 12h | Fatura vence hoje | "Fatura [cartão] vence hoje: R$ X. Pagar agora?" |
| 14h | Hábito flexível em risco semanal | "Faltam 2 sessões de [hábito]. Hoje e amanhã são suas últimas chances." |
| 18h | Resumo do dia até agora | "5/7 hábitos feitos. Faltam: Leitura, Journaling. Treino: ✓ (rating 4)." |
| 21h | Planejamento do dia seguinte | "Amanhã: 3 eventos, 5 hábitos, 1 conta a pagar. Dia tranquilo." |

### Inteligência de timing
- Não notifica se o usuário já fez a ação
- Não repete notificação ignorada (backoff)
- Aprende quais notificações geram ação (via Langfuse feedback) e prioriza

### Output
Inicialmente via endpoint `GET /agents/notifications` (frontend faz polling).
Futuro: webhook para push notifications reais.

### Complexidade: Média
Lógica de regras + timing + anti-spam. Foundation para mobile app futuro.

---

## 20. Messaging Gateway Agent (Telegram / WhatsApp / SMS)

**Trigger:** Webhook recebido de plataforma de mensageria

O usuário envia uma mensagem de texto pelo Telegram (ou WhatsApp, SMS) e o agente interpreta, cria o lançamento no backend correto e responde com confirmação — tudo sem abrir o app.

### O problema
Lançar uma despesa ou criar um evento requer abrir o app, navegar até o formulário, preencher campos. Na prática, o usuário esquece de registrar gastos pequenos e eventos rápidos porque a fricção é alta demais. Uma mensagem de texto é o input mais rápido que existe.

### Arquitetura

```
[Telegram/WhatsApp]
    → Webhook (POST /webhooks/telegram ou /webhooks/whatsapp)
    → Messaging Gateway Agent
        → Identifica usuário (chat_id → profileId/calendarId)
        → Delega para Natural Language Creator (Agent 2)
        → Recebe resultado estruturado
        → Responde no mesmo canal com confirmação
```

O Messaging Gateway é uma **camada fina** sobre o Natural Language Creator — não duplica lógica de parsing, apenas gerencia o canal de comunicação.

### Fluxo detalhado

```
Usuário (Telegram): "Almoco 32 nubank"

Gateway Agent:
  1. Recebe webhook com chat_id + texto
  2. Resolve chat_id → profileId (tabela de binding)
  3. Passa texto para Agent 2 (NL Creator)
  4. Agent 2 retorna:
     {
       "domain": "finances",
       "action": "create_transaction",
       "data": {
         "type": "EXPENSE",
         "status": "CONFIRMED",
         "description": "Almoço",
         "amount": 32.00,
         "bankAccount": "Nubank" (resolved by name),
         "category": "Alimentação" (auto-categorized)
       }
     }
  5. Executa POST /transactions no calendar-finances
  6. Responde no Telegram:
     "✓ Despesa registrada: Almoço R$ 32,00 (Nubank → Alimentação)"
```

### Exemplos de mensagens suportadas

| Mensagem | Ação | Resposta |
|----------|------|----------|
| "almoco 32 nubank" | Cria EXPENSE R$ 32 no Nubank | "✓ Almoço R$ 32,00 (Nubank → Alimentação)" |
| "uber 18.50" | Cria EXPENSE R$ 18,50 (conta default) | "✓ Uber R$ 18,50 (Itaú → Transporte)" |
| "salario 8000" | Cria INCOME R$ 8.000 | "✓ Salário R$ 8.000,00 (Itaú → Renda)" |
| "reuniao joao amanha 14h" | Cria EVENT no calendar-core | "✓ Reunião com João — amanhã 14:00" |
| "paguei internet" | Confirma recurring transaction "Internet" | "✓ Internet R$ 120,00 marcada como paga" |
| "quanto gastei hoje" | Consulta (delega para Agent 18) | "Hoje: R$ 87,50 em 3 transações" |
| "streak" | Consulta habits/stats | "Meditação: 23 dias 🔥 / Exercício: 8 dias" |
| "saldo" | Consulta bank-accounts | "Itaú: R$ 4.520 / Nubank: fatura R$ 1.230" |
| "/undo" | Desfaz último lançamento | "↩ Desfeito: Uber R$ 18,50" |
| "/resumo" | Daily Briefing condensado (Agent 1) | Resumo do dia em texto |

### Funcionalidades do canal

#### 20.1 Binding de usuário
- Primeiro acesso: usuário envia `/start` no Telegram
- Bot responde com link de autenticação (JWT temporário)
- Após auth, persiste `chat_id → userId/profileId/calendarId`
- Suporte a múltiplos perfis: `/perfil pessoal` ou `/perfil business`

#### 20.2 Confirmação inline
Para lançamentos ambíguos, o bot pergunta antes de criar:
```
Usuário: "50 reais"
Bot: "R$ 50,00 — é uma despesa? Qual conta?"
     [Nubank] [Itaú] [Cancelar]
```
No Telegram: usa InlineKeyboard. No WhatsApp: usa botões ou resposta numérica.

#### 20.3 Modo rápido vs. modo completo
- **Rápido** (default): mínimo de campos, auto-resolve o resto
  - "uber 18" → EXPENSE, CONFIRMED, conta default, categoria auto
- **Completo**: quando o usuário quer especificar tudo
  - "uber 18 nubank transporte cartao" → conta Nubank, categoria Transporte, via cartão de crédito

#### 20.4 Contexto de conversa
Mantém contexto das últimas mensagens para follow-ups:
```
Usuário: "almoco 45"
Bot: "✓ Almoço R$ 45,00 (Nubank → Alimentação)"
Usuário: "foi no ifood"
Bot: "✓ Atualizado: Almoço → Alimentação > Delivery"
```

#### 20.5 Comandos especiais

| Comando | Ação |
|---------|------|
| `/start` | Inicia binding com o sistema |
| `/perfil` | Troca perfil ativo (pessoal/business) |
| `/conta` | Define conta default para lançamentos rápidos |
| `/resumo` | Daily Briefing condensado |
| `/saldo` | Saldos de todas as contas |
| `/streak` | Status dos streaks de hábitos |
| `/undo` | Desfaz último lançamento |
| `/help` | Lista de comandos e exemplos |

### Plataformas

#### Telegram (prioridade 1)
- Bot API nativa, gratuita, sem aprovação
- InlineKeyboard para confirmações
- Suporte a grupos (family finance)
- Webhook direto para FastAPI

#### WhatsApp (prioridade 2)
- WhatsApp Business API (via Twilio ou Meta Cloud API)
- Requer aprovação de template para mensagens proativas
- Botões limitados (máx 3)
- Custo por mensagem após 24h window

#### SMS (prioridade 3)
- Via Twilio
- Sem botões, apenas texto
- Útil como fallback
- Custo por mensagem

### Implementação no agents

```
services/agents/
└── app/
    ├── webhooks/
    │   ├── telegram.py      # POST /webhooks/telegram (webhook handler)
    │   └── whatsapp.py      # POST /webhooks/whatsapp (futuro)
    ├── messaging/
    │   ├── gateway.py        # Messaging Gateway Agent (orquestrador)
    │   ├── binding.py        # User ↔ chat_id binding
    │   ├── context.py        # Conversation context (últimas N mensagens)
    │   └── responder.py      # Formata respostas para cada plataforma
    └── graph.py              # LangGraph: inclui nó de messaging no grafo
```

### Integração com outros agentes

O Messaging Gateway não é um agente isolado — é um **canal de entrada** para todos os outros:

| Mensagem | Agente delegado |
|----------|----------------|
| "almoco 32 nubank" | Agent 2 (NL Creator) |
| "quanto gastei hoje" | Agent 18 (Conversational Report) |
| "/resumo" | Agent 1 (Daily Briefing) |
| "streak" | Agent 3 (Habit Coach) |
| "consigo juntar 5000 ate junho?" | Agent 10 (Financial Forecast) |
| "score" | Agent 17 (Gamification) |

E é também o **canal de saída** para notificações proativas:

| Agente | Notificação via Telegram |
|--------|--------------------------|
| Agent 19 (Proactive Notifications) | "Fatura Nubank vence amanhã: R$ 2.340" |
| Agent 3 (Habit Coach) | "Streak de meditação em risco — 2 dias sem completar" |
| Agent 4 (Financial Guardian) | "Alimentação 40% acima da média este mês" |

### Complexidade: Média
Webhook handler + thin layer sobre Agent 2. A complexidade real está nos agentes delegados. Telegram Bot API é simples e bem documentada.

---

## Priorização atualizada (todos os 20 agentes)

### Fase 1 — Foundation (valor imediato, baixa complexidade)
| # | Agent | Valor | Complexidade |
|---|-------|-------|-------------|
| 1 | Daily Briefing | Alto | Baixa |
| 5 | Smart Categorization | Médio | Baixa |
| 12 | Expense Negotiator | Médio | Baixa |

### Fase 2 — Core Intelligence (alto valor, complexidade moderada)
| # | Agent | Valor | Complexidade |
|---|-------|-------|-------------|
| 2 | Natural Language Creator | Alto | Média |
| 3 | Habit Coach | Alto | Média |
| 4 | Financial Guardian | Alto | Média |
| 10 | Financial Forecast | Alto | Média |
| 17 | Gamification & Scoring | Médio | Média |
| 19 | Proactive Notifications | Alto | Média |
| 20 | Messaging Gateway (Telegram) | Alto | Média |

### Fase 3 — Advanced Analytics
| # | Agent | Valor | Complexidade |
|---|-------|-------|-------------|
| 6 | Weekly Review | Médio | Média |
| 11 | Habit Stacking | Médio | Média |
| 13 | Sleep & Recovery Inference | Médio | Média |
| 16 | Import & Reconciliation | Médio | Média |
| 18 | Conversational Report | Alto | Média-Alta |

### Fase 4 — Deep Intelligence
| # | Agent | Valor | Complexidade |
|---|-------|-------|-------------|
| 7 | Cross-Domain Correlation | Alto | Alta |
| 8 | Time Architect | Alto | Alta |
| 9 | Accountability Partner | Alto | Alta |
| 14 | Goal Decomposition | Alto | Alta |
| 15 | Anomaly & Life Event Detector | Alto | Alta |
