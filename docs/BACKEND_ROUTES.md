# Backend API Routes - Calendar App

## 📋 Estrutura de Rotas NestJS

### Authentication & Users
```
POST   /auth/register          # Criar conta
POST   /auth/login             # Login
POST   /auth/logout            # Logout
GET    /auth/me                # Dados do usuário logado
POST   /auth/google/callback   # OAuth2 Google Calendar
```

### Calendars (Agendas)
```
GET    /calendars              # Listar calendários do usuário
GET    /calendars/:id          # Detalhes de um calendário
POST   /calendars              # Criar calendário
PUT    /calendars/:id          # Atualizar calendário
DELETE /calendars/:id          # Remover calendário
```

**Payload Example**:
```json
{
  "name": "WB Digital Solutions",
  "email": "bruno@wbdigitalsolutions.com",
  "color": "#350545",
  "type": "professional",
  "googleCalendarId": "primary"
}
```

---

### Categories (Categorias)
```
GET    /categories             # Listar todas as categorias
GET    /categories/:id         # Detalhes de uma categoria
POST   /categories             # Criar categoria
PUT    /categories/:id         # Atualizar categoria
DELETE /categories/:id         # Remover categoria
```

**Payload Example**:
```json
{
  "calendarId": "uuid",
  "name": "Academia",
  "icon": "🏃",
  "color": "#FF5733",
  "type": "health"
}
```

---

### Events (Eventos)
```
GET    /events                 # Listar eventos (com filtros)
GET    /events/:id             # Detalhes de um evento
POST   /events                 # Criar evento
PUT    /events/:id             # Atualizar evento
DELETE /events/:id             # Remover evento
GET    /events/date/:date      # Eventos de uma data específica
GET    /events/range           # Eventos em um período
```

**Query Params (GET /events)**:
- `calendarId`: Filtrar por calendário
- `categoryId`: Filtrar por categoria
- `startDate`: Data inicial
- `endDate`: Data final
- `completed`: true/false (eventos concluídos)

**Payload Example (POST)**:
```json
{
  "calendarId": "uuid",
  "categoryId": "uuid",
  "title": "Academia - Treino A",
  "description": "Peito, ombro e tríceps",
  "startTime": "18:00",
  "endTime": "19:00",
  "isRecurring": true,
  "recurrencePattern": {
    "frequency": "weekly",
    "interval": 1,
    "daysOfWeek": [1, 3, 5],
    "endDate": null
  },
  "reminders": [
    { "minutesBefore": 30, "method": "notification" }
  ]
}
```

---

### Event Executions (Marcação de Execução)
```
POST   /events/:id/executions        # Marcar evento como feito/não feito
GET    /events/:id/executions        # Histórico de execuções do evento
GET    /executions/date/:date        # Todas as execuções de uma data
GET    /executions/stats             # Estatísticas de execução
```

**Payload Example**:
```json
{
  "eventId": "uuid",
  "executionDate": "2025-10-10",
  "completed": true,
  "notes": "Treino completo, adicionei 5kg no supino"
}
```

---

### Monthly Goals (Metas Mensais)
```
GET    /goals                  # Listar metas
GET    /goals/:id              # Detalhes de uma meta
POST   /goals                  # Criar meta mensal
PUT    /goals/:id              # Atualizar meta
DELETE /goals/:id              # Remover meta
GET    /goals/category/:id     # Metas de uma categoria
GET    /goals/month/:yearMonth # Metas de um mês específico
```

**Payload Example**:
```json
{
  "categoryId": "uuid",
  "month": "2025-10",
  "targetExecutions": 12,
  "description": "Academia 3x por semana"
}
```

---

### Reports (Relatórios)
```
GET    /reports/monthly/:yearMonth   # Relatório mensal completo
GET    /reports/category/:id         # Relatório de uma categoria
GET    /reports/compare              # Comparação entre meses
POST   /reports/generate             # Gerar relatório (AI)
```

**Response Example (GET /reports/monthly/2025-10)**:
```json
{
  "month": "2025-10",
  "summary": {
    "totalEvents": 120,
    "completedEvents": 95,
    "completionRate": 79.17,
    "categoriesBreakdown": [
      {
        "categoryId": "uuid",
        "categoryName": "Academia",
        "total": 12,
        "completed": 10,
        "rate": 83.33
      }
    ]
  },
  "insights": [
    "Você melhorou 15% em Saúde comparado ao mês passado",
    "Categoria Meditação está abaixo da meta (50%)"
  ]
}
```

---

### Google Calendar Sync
```
POST   /sync/google/import     # Importar eventos do Google Calendar
POST   /sync/google/export     # Exportar eventos para Google Calendar
GET    /sync/google/status     # Status da última sincronização
POST   /sync/google/manual     # Forçar sincronização manual
```

---

### Statistics & Analytics
```
GET    /stats/daily/:date      # Estatísticas de um dia
GET    /stats/weekly           # Estatísticas da semana
GET    /stats/monthly/:month   # Estatísticas do mês
GET    /stats/category/:id     # Stats de uma categoria específica
GET    /stats/trends           # Tendências ao longo do tempo
```

---

## 🗂️ Estrutura de Pastas NestJS (DDD)

```
services/calendar-core/src/
├── domains/
│   ├── auth/
│   │   ├── domain/
│   │   │   └── entities/
│   │   ├── application/
│   │   │   └── use-cases/
│   │   └── infrastructure/
│   │       ├── controllers/
│   │       ├── repositories/
│   │       └── services/
│   ├── calendars/
│   │   ├── domain/entities/
│   │   ├── application/use-cases/
│   │   └── infrastructure/
│   ├── categories/
│   ├── events/
│   ├── executions/
│   ├── goals/
│   ├── reports/
│   └── sync/
├── shared/
│   ├── database/
│   ├── guards/
│   └── decorators/
└── app.module.ts
```

---

## 🔐 Autenticação

Todas as rotas (exceto `/auth/register` e `/auth/login`) requerem:
- **JWT Token** no header: `Authorization: Bearer <token>`

---

## 📊 Padrões de Recorrência

### Frequency Types:
- `daily`: Diário
- `weekly`: Semanal
- `monthly`: Mensal
- `yearly`: Anual

### Days of Week (para weekly):
- `0` = Domingo
- `1` = Segunda
- `2` = Terça
- `3` = Quarta
- `4` = Quinta
- `5` = Sexta
- `6` = Sábado

### Example Patterns:

**Academia 3x/semana (Seg, Qua, Sex)**:
```json
{
  "frequency": "weekly",
  "interval": 1,
  "daysOfWeek": [1, 3, 5]
}
```

**Meditação diária**:
```json
{
  "frequency": "daily",
  "interval": 1
}
```

**Reunião mensal (primeira segunda do mês)**:
```json
{
  "frequency": "monthly",
  "interval": 1,
  "dayOfMonth": 1,
  "weekOfMonth": 1
}
```

---

## 🎯 Próximos Passos de Implementação

### Fase 1: MVP Backend
1. ✅ Setup NestJS com TypeORM/Prisma
2. ⏳ Criar entidades e migrations
3. ⏳ Implementar rotas de Calendars
4. ⏳ Implementar rotas de Categories
5. ⏳ Implementar rotas de Events
6. ⏳ Implementar sistema de Executions
7. ⏳ Implementar Goals
8. ⏳ Implementar Reports básicos

### Fase 2: Integrações
- Google Calendar OAuth2
- Sincronização bidirecional
- Webhooks para atualizações automáticas

### Fase 3: Análise
- AI insights com Python service
- Geração de relatórios avançados
- Recomendações personalizadas
