# Plano: Integração Google Calendar (Bidirecional)

## Contexto

O sistema já tem `Calendar.googleCalendarId` e `Event.googleEventId` no schema (ambos indexados), e o domínio `google-calendar/` existe como stub. A integração é **opcional por calendário** — se `googleCalendarId` for null, o calendário funciona normalmente sem nenhuma interação com o Google.

**Contas a integrar:**
- Calendário pessoal → `wbrunovieira77@gmail.com`
- Calendário WB Digital Solutions → `bruno@wbdigitalsolutions.com` *(OAuth credentials já existentes no CRM)*

**Direção do sync:**
- Local → Google: eventos criados/editados/deletados aqui são espelhados no Google Calendar
- Google → Local: polling a cada 5 min com `syncToken` incremental busca alterações feitas diretamente no Google Calendar

---

## Arquitetura da Solução

```
google-calendar/
├── domain/
│   ├── entities/
│   │   ├── calendar-event.entity.ts    (já existe, stub)
│   │   └── google-oauth-token.entity.ts  (novo)
│   └── services/
│       └── google-event-mapper.ts      (converte Event ↔ Google Event)
├── application/
│   └── use-cases/
│       ├── connect-google-calendar.use-case.ts
│       ├── disconnect-google-calendar.use-case.ts
│       ├── trigger-full-sync.use-case.ts
│       ├── sync-event-to-google.use-case.ts
│       └── pull-google-changes.use-case.ts
└── infrastructure/
    ├── controllers/
    │   └── google-calendar.controller.ts
    ├── dtos/
    ├── repositories/
    │   └── google-oauth-token.repository.ts (interface)
    ├── persistence/
    │   └── prisma-google-oauth-token.repository.ts
    └── services/
        ├── google-calendar-api.client.ts   (wrapper da googleapis)
        └── google-calendar-polling.service.ts  (@Cron)
```

**Prevenção de loop de sync:** campo `syncSource: 'google' | 'internal'` no contexto da operação — quando o polling do Google cria/atualiza um evento local, esse flag impede a propagação de volta ao Google.

---

## Etapa 1 — Schema: Token OAuth e campos de sync

**Branch:** `feat/google-calendar-foundation`  
**Commit:** `feat(google-calendar): add OAuth token storage and sync state to schema`

### O que fazer

**1. Novo model no `schema.prisma`:**

```prisma
model GoogleOAuthToken {
  id           String   @id @default(uuid())
  email        String   @unique  // Google account email — chave de lookup
  accessToken  String   @map("access_token") @db.Text
  refreshToken String   @map("refresh_token") @db.Text
  expiresAt    DateTime @map("expires_at")
  createdAt    DateTime @default(now())
  updatedAt    DateTime @updatedAt
}
```

**2. Campos adicionais no model `Calendar`:**

```prisma
googleSyncToken String?   @map("google_sync_token") @db.Text  // syncToken incremental do Google
lastSyncAt      DateTime? @map("last_sync_at")
```

**3. Gerar migration:**

```bash
docker-compose exec calendar-core npx prisma migrate dev --name "add_google_oauth_token_and_calendar_sync_state"
docker-compose exec calendar-core npx prisma generate
```

**4. Atualizar `Calendar` entity** com os novos campos opcionais.

### Testes

- Verificar migração idempotente no banco de teste
- Unit test: `Calendar.create()` aceita e ignora `googleSyncToken`/`lastSyncAt` quando null

### Deploy notes

Nenhum. Migration aplica sem breaking change (todos os campos são nullable).

---

## Etapa 2 — OAuth Flow (autorização e tokens)

**Branch:** `feat/google-calendar-oauth`  
**Commit:** `feat(google-calendar): OAuth authorization flow and token management`

### O que fazer

**Endpoints:**

```
GET  /google-calendar/auth/connect?calendarId=:id
     → Redireciona para OAuth do Google com state=calendarId

GET  /google-calendar/auth/callback?code=:code&state=:calendarId
     → Troca code por tokens, armazena em GoogleOAuthToken por email,
       atualiza Calendar.googleCalendarId = 'primary' (ou o ID real),
       redireciona para o frontend com sucesso/erro

DELETE /google-calendar/auth/disconnect/:calendarId
     → Revoga token no Google, remove GoogleOAuthToken, limpa
       Calendar.googleCalendarId + googleSyncToken + lastSyncAt
```

**`GoogleOAuthService`** (infrastructure/services):

```typescript
async getValidAccessToken(email: string): Promise<string>
// Verifica expiresAt, faz refresh automático se necessário:
// POST https://oauth2.googleapis.com/token { grant_type: refresh_token }
// Atualiza accessToken + expiresAt no banco

async exchangeCode(code: string): Promise<TokenData>
async revokeToken(email: string): Promise<void>
```

**Refresh automático:** buffer de 5 minutos antes de `expiresAt`.

**Scopes necessários:**
```
https://www.googleapis.com/auth/calendar
https://www.googleapis.com/auth/calendar.events
https://www.googleapis.com/auth/userinfo.email
```

**Env vars a adicionar no `.env.example`:**

```
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
GOOGLE_REDIRECT_URI=http://localhost:3334/google-calendar/auth/callback
GOOGLE_FRONTEND_REDIRECT_URI=http://localhost:3000/settings/integrations
```

**Para produção:** adicionar `https://calendar-api.wbdigitalsolutions.com/google-calendar/auth/callback` no Google Cloud Console (projeto 154238465749).

### Instalação

```bash
docker-compose exec calendar-core npm install googleapis
```

### TDD

```
google-oauth.service.spec.ts
  ✓ retorna access token válido sem refresh quando não expirado
  ✓ faz refresh automático quando expiresAt < 5 min
  ✓ persiste novo accessToken + expiresAt após refresh
  ✓ lança GoogleNotConnectedException quando email não tem token
  ✓ revoga token e remove do banco no disconnect

connect-google-calendar.use-case.spec.ts
  ✓ salva token e atualiza Calendar.googleCalendarId após callback bem-sucedido
  ✓ lança CalendarNotFoundException se calendarId inválido

disconnect-google-calendar.use-case.spec.ts
  ✓ limpa googleCalendarId, googleSyncToken, lastSyncAt no calendário
```

### Deploy notes

- Adicionar `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GOOGLE_REDIRECT_URI` no `.env` de produção (via Ansible vault)
- Adicionar redirect URI no Google Cloud Console
- Para a conta pessoal (`wbrunovieira77@gmail.com`): novo fluxo OAuth com o mesmo app (já tem scopes aprovados)

---

## Etapa 3 — Google Calendar API Client e Mapper

**Branch:** `feat/google-calendar-client`  
**Commit:** `feat(google-calendar): Google Calendar API client and event mapper`

### O que fazer

**`GoogleCalendarApiClient`** (infrastructure/services):

```typescript
async createEvent(googleCalendarId: string, event: GoogleEventPayload): Promise<string>
// retorna googleEventId criado

async updateEvent(googleCalendarId: string, googleEventId: string, event: Partial<GoogleEventPayload>): Promise<void>

async deleteEvent(googleCalendarId: string, googleEventId: string): Promise<void>

async fetchChanges(googleCalendarId: string, syncToken?: string): Promise<{
  events: GoogleEvent[]
  nextSyncToken: string
}>
// syncToken null = full sync; com syncToken = incremental (só mudanças)

async listAllEvents(googleCalendarId: string): Promise<GoogleEvent[]>
// usado apenas no sync inicial
```

Todas as chamadas chamam `oauthService.getValidAccessToken(email)` antes de executar.

**`GoogleEventMapper`** (domain/services):

```typescript
toGoogleEvent(event: Event): GoogleEventPayload
// Mapeia: title, description, startDate+startTime, endDate+endTime,
//         recurrenceRule (RRULE), reminders (minutesBefore → overrides),
//         allDay (endTime null)

toLocalEvent(googleEvent: GoogleEvent, calendarId: string): Partial<Event>
// Inverso: extrai googleEventId, título, datas, RRULE, etc.
// Campos sem equivalente local (attendees, location) são ignorados
```

**Casos de mapeamento não-óbvios:**
- Evento all-day no Google: `start: { date: 'YYYY-MM-DD' }` (sem `dateTime`)
- RRULE: Google aceita array `recurrence: ['RRULE:FREQ=WEEKLY;...']`
- Fuso horário: sempre serializar em `America/Sao_Paulo`

### TDD

```
google-event-mapper.spec.ts
  ✓ mapeia evento simples (local → Google)
  ✓ mapeia evento recorrente preservando RRULE
  ✓ mapeia evento all-day
  ✓ mapeia reminders para formato Google overrides
  ✓ mapeia evento Google → local preservando googleEventId
  ✓ ignora campos Google sem equivalente local (attendees, location)
  ✓ mapeia evento cancelado do Google como status CANCELLED
```

Usar mocks para o `GoogleCalendarApiClient` — zero chamadas reais nos testes.

---

## Etapa 4 — Sync de Saída (local → Google)

**Branch:** `feat/google-calendar-outbound-sync`  
**Commit:** `feat(google-calendar): sync local event mutations to Google Calendar`

### O que fazer

**`GoogleCalendarSyncService`** (application use-case orquestrador):

```typescript
async onEventCreated(event: Event, calendarId: string): Promise<void>
async onEventUpdated(event: Event, scope: RecurringEditScope): Promise<void>
async onEventDeleted(eventId: string, googleEventId: string, calendarId: string): Promise<void>
```

Cada método verifica se `calendar.googleCalendarId != null` antes de agir. Se o calendário não tem Google Calendar conectado, retorna silenciosamente (feature opcional).

**Integração nos use-cases existentes** via injeção direta (não EventEmitter para manter simplicidade):

```typescript
// CreateEventUseCase — após salvar no banco:
if (calendar.googleCalendarId) {
  const googleId = await this.googleSync.onEventCreated(event, calendar)
  await this.eventRepo.updateGoogleEventId(event.id, googleId)
}

// UpdateEventUseCase — após salvar:
if (event.googleEventId) {
  await this.googleSync.onEventUpdated(event, input.recurringEditScope)
}

// DeleteEventUseCase — antes de deletar:
if (event.googleEventId) {
  await this.googleSync.onEventDeleted(event.id, event.googleEventId, calendar)
}
```

**Falhas na API do Google são não-fatais:** log de erro + evento local salvo normalmente. O sync pode ser reenviado via endpoint manual de re-sync.

**Escopo de recorrência no Google:**
- `'this'` → `updateEvent` com `recurringEventId` (modifica instância)
- `'future'` → cria nova série no Google
- `'all'` → `updateEvent` no evento master

### TDD

```
sync-event-to-google.use-case.spec.ts
  ✓ cria evento no Google e persiste googleEventId
  ✓ não chama API quando googleCalendarId é null (opcional)
  ✓ atualiza evento existente no Google
  ✓ deleta evento no Google
  ✓ falha na API não impede persistência local (log + continua)
  ✓ scope 'this' chama update de instância única
  ✓ scope 'all' chama update do evento master
```

---

## Etapa 5 — Sync de Entrada (Google → local)

**Branch:** `feat/google-calendar-inbound-sync`  
**Commit:** `feat(google-calendar): poll Google Calendar for inbound changes`

### O que fazer

**`GoogleCalendarPollingService`** (NestJS `@Cron`):

```typescript
@Cron('*/5 * * * *')  // a cada 5 minutos
async pollAllConnectedCalendars(): Promise<void>
  // Para cada Calendar onde googleCalendarId != null AND lastSyncAt != null:
  //   1. fetchChanges(googleCalendarId, googleSyncToken)
  //   2. Para cada evento retornado:
  //      - status 'cancelled' → deletar evento local (se googleEventId existe)
  //      - googleEventId existe no banco → updateEvent (com syncSource='google')
  //      - não existe → createEvent (com syncSource='google')
  //   3. Atualizar googleSyncToken + lastSyncAt no Calendar
```

**Prevenção de loop:** `syncSource` é um parâmetro passado nos use-cases internos:

```typescript
// Em CreateEventUseCase e UpdateEventUseCase:
if (input.syncSource === 'google') return  // não propaga de volta ao Google
```

**`PullGoogleChangesUseCase`** — pode ser chamado também manualmente:
```
POST /google-calendar/sync/:calendarId/pull
```

**`TriggerFullSyncUseCase`:**
```
POST /google-calendar/sync/:calendarId/full
```
- Importa todos os eventos via `listAllEvents`
- Para cada evento: upsert por `googleEventId`
- Define `googleSyncToken` para futuros pulls incrementais
- Define `lastSyncAt`

### TDD

```
pull-google-changes.use-case.spec.ts
  ✓ cria evento local para novo evento Google
  ✓ atualiza evento local para evento Google modificado
  ✓ deleta evento local para evento Google cancelado
  ✓ persiste nextSyncToken no Calendar após sync
  ✓ não propaga de volta ao Google (syncSource='google')
  ✓ ignora erros de API e continua (não quebra o cron)

google-calendar-polling.service.spec.ts
  ✓ processa apenas calendários com googleCalendarId não-null
  ✓ usa syncToken incremental quando disponível
  ✓ faz full list quando syncToken é null (primeira vez)
```

### Instalação adicional no NestJS

```typescript
// app.module.ts
imports: [
  ScheduleModule.forRoot(),  // @nestjs/schedule
  GoogleCalendarModule,
]
```

```bash
docker-compose exec calendar-core npm install @nestjs/schedule
```

---

## Etapa 6 — Frontend: Settings de Integração

**Branch:** `feat/google-calendar-frontend-settings`  
**Commit:** `feat(google-calendar): add Google Calendar connection UI to settings`

### O que fazer

Tela em `services/calendar-frontend/` em `/settings/integrations`:

- Lista calendários existentes
- Para cada calendário: botão "Conectar Google Calendar" → abre OAuth flow
- Calendários conectados: mostra o email Google vinculado + botão "Desconectar"
- Status de último sync (`lastSyncAt`)
- Botão "Sincronizar agora" → `POST /google-calendar/sync/:id/pull`

---

## Checklist de Deploy (produção)

1. **Google Cloud Console** (projeto 154238465749):
   - Adicionar redirect URI: `https://calendar-api.wbdigitalsolutions.com/google-calendar/auth/callback`
   - Confirmar scopes `calendar` e `calendar.events` no app OAuth

2. **Ansible vault** — adicionar ao `.env` de produção:
   ```
   GOOGLE_CLIENT_ID=<your-google-client-id>
   GOOGLE_CLIENT_SECRET=<your-google-client-secret>
   GOOGLE_REDIRECT_URI=https://calendar-api.wbdigitalsolutions.com/google-calendar/auth/callback
   GOOGLE_FRONTEND_REDIRECT_URI=https://calendar.wbdigitalsolutions.com/settings/integrations
   ```

3. **Primeiro uso:** acessar Settings → Integrations, clicar "Conectar" para cada calendário, logar com a conta Google correspondente.

4. **Sync inicial:** após conectar, clicar "Sincronizar agora" (ou aguardar o próximo cron de 5 min). O `lastSyncAt` sendo null dispara full sync automaticamente na primeira execução do cron.

---

## Ordem de implementação recomendada

| # | Etapa | Dependências |
|---|-------|-------------|
| 1 | Schema (migration) | — |
| 2 | OAuth flow | Etapa 1 |
| 3 | API Client + Mapper | Etapa 2 |
| 4 | Sync de saída | Etapa 3 |
| 5 | Sync de entrada | Etapas 3 + 4 |
| 6 | Frontend settings | Etapas 2–5 |

As etapas 2–5 podem ser desenvolvidas em paralelo uma vez que o schema está no ar, pois cada uma tem testes independentes com mocks.
