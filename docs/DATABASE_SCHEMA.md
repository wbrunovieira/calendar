# Database Schema - Calendar App

## 🗄️ PostgreSQL Database Structure

### Entity Relationship Diagram (ERD)

```
users (1) ─────< (N) calendars
                      │
                      └─< (N) categories
                               │
                               ├─< (N) events
                               │    │
                               │    ├─< (N) event_executions
                               │    └─< (N) event_reminders
                               │
                               └─< (N) monthly_goals
```

---

## 📊 Tables Definition

### 1. **users**
Usuários do sistema

```sql
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email VARCHAR(255) UNIQUE NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  name VARCHAR(255) NOT NULL,
  avatar_url VARCHAR(500),
  timezone VARCHAR(50) DEFAULT 'America/Sao_Paulo',
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);
```

**Indexes**:
- `idx_users_email` ON email

---

### 2. **calendars**
Calendários (WB Digital Solutions, Pessoal)

```sql
CREATE TABLE calendars (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  email VARCHAR(255),
  color VARCHAR(7) NOT NULL, -- HEX color (#350545)
  type VARCHAR(50) NOT NULL, -- 'professional' | 'personal'
  google_calendar_id VARCHAR(255),
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);
```

**Indexes**:
- `idx_calendars_user_id` ON user_id
- `idx_calendars_google_id` ON google_calendar_id

---

### 3. **categories**
Categorias de eventos (Academia, Reuniões, etc)

```sql
CREATE TABLE categories (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  calendar_id UUID NOT NULL REFERENCES calendars(id) ON DELETE CASCADE,
  name VARCHAR(255) NOT NULL,
  icon VARCHAR(10), -- Emoji (🏃, 💼)
  color VARCHAR(7) NOT NULL, -- HEX color
  type VARCHAR(50), -- 'health', 'work', 'leisure', etc
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);
```

**Indexes**:
- `idx_categories_calendar_id` ON calendar_id

---

### 4. **events**
Eventos (recorrentes ou únicos)

```sql
CREATE TABLE events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  calendar_id UUID NOT NULL REFERENCES calendars(id) ON DELETE CASCADE,
  category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
  title VARCHAR(255) NOT NULL,
  description TEXT,
  start_time TIME NOT NULL,
  end_time TIME,
  start_date DATE,
  end_date DATE,

  -- Recorrência
  is_recurring BOOLEAN DEFAULT false,
  recurrence_frequency VARCHAR(20), -- 'daily', 'weekly', 'monthly', 'yearly'
  recurrence_interval INT DEFAULT 1,
  recurrence_days_of_week INT[], -- [0,1,2,3,4,5,6]
  recurrence_day_of_month INT,
  recurrence_week_of_month INT,
  recurrence_end_date DATE,

  -- Google Calendar
  google_event_id VARCHAR(255),

  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);
```

**Indexes**:
- `idx_events_calendar_id` ON calendar_id
- `idx_events_category_id` ON category_id
- `idx_events_start_date` ON start_date
- `idx_events_google_id` ON google_event_id

---

### 5. **event_executions**
Registro de execução diária (feito/não feito)

```sql
CREATE TABLE event_executions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  execution_date DATE NOT NULL,
  completed BOOLEAN DEFAULT false,
  completed_at TIMESTAMP,
  notes TEXT,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),

  UNIQUE(event_id, execution_date)
);
```

**Indexes**:
- `idx_executions_event_id` ON event_id
- `idx_executions_date` ON execution_date
- `idx_executions_completed` ON completed

---

### 6. **event_reminders**
Lembretes de eventos

```sql
CREATE TABLE event_reminders (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
  minutes_before INT NOT NULL,
  method VARCHAR(50) NOT NULL, -- 'email', 'notification', 'sms'
  created_at TIMESTAMP DEFAULT NOW()
);
```

**Indexes**:
- `idx_reminders_event_id` ON event_id

---

### 7. **monthly_goals**
Metas mensais por categoria

```sql
CREATE TABLE monthly_goals (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
  month DATE NOT NULL, -- Primeiro dia do mês (2025-10-01)
  target_executions INT NOT NULL,
  description TEXT,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),

  UNIQUE(category_id, month)
);
```

**Indexes**:
- `idx_goals_category_id` ON category_id
- `idx_goals_month` ON month

---

### 8. **reports**
Relatórios mensais gerados (cache)

```sql
CREATE TABLE reports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  month DATE NOT NULL,
  data JSONB NOT NULL, -- Relatório completo em JSON
  ai_insights TEXT[],
  created_at TIMESTAMP DEFAULT NOW(),

  UNIQUE(user_id, month)
);
```

**Indexes**:
- `idx_reports_user_id` ON user_id
- `idx_reports_month` ON month

---

## 📐 Example Data

### Seed Data - Categories (WB Digital Solutions)

```sql
INSERT INTO categories (calendar_id, name, icon, color, type) VALUES
  ('calendar-uuid', 'Reuniões', '💼', '#FF5733', 'work'),
  ('calendar-uuid', 'Prospecção', '📞', '#33A1FF', 'work'),
  ('calendar-uuid', 'Programação', '💻', '#33FF57', 'work'),
  ('calendar-uuid', 'Financeiro', '💰', '#FFD700', 'work');
```

### Seed Data - Categories (Pessoal)

```sql
INSERT INTO categories (calendar_id, name, icon, color, type) VALUES
  ('calendar-uuid', 'Academia', '🏃', '#FF6B6B', 'health'),
  ('calendar-uuid', 'Jiu-Jitsu', '🥋', '#4ECDC4', 'health'),
  ('calendar-uuid', 'Natação', '🏊', '#1A535C', 'health'),
  ('calendar-uuid', 'Meditação', '🧘', '#9B59B6', 'wellness'),
  ('calendar-uuid', 'Leitura', '📖', '#3498DB', 'leisure');
```

### Example Event - Academia (Recorrente)

```sql
INSERT INTO events (
  calendar_id, category_id, title, description,
  start_time, end_time,
  is_recurring, recurrence_frequency, recurrence_interval, recurrence_days_of_week
) VALUES (
  'calendar-uuid', 'category-uuid',
  'Academia - Treino A',
  'Peito, ombro e tríceps',
  '18:00', '19:00',
  true, 'weekly', 1, ARRAY[1, 3, 5]
);
```

### Example Execution - Marcar como feito

```sql
INSERT INTO event_executions (event_id, execution_date, completed, notes)
VALUES (
  'event-uuid',
  '2025-10-10',
  true,
  'Treino completo! Aumentei 5kg no supino.'
);
```

---

## 🔍 Queries Úteis

### Listar eventos do dia com status de execução

```sql
SELECT
  e.id,
  e.title,
  e.start_time,
  c.name as category_name,
  c.icon,
  c.color,
  COALESCE(ex.completed, false) as completed
FROM events e
LEFT JOIN categories c ON e.category_id = c.id
LEFT JOIN event_executions ex ON e.id = ex.event_id
  AND ex.execution_date = '2025-10-10'
WHERE e.calendar_id = 'calendar-uuid'
  AND (
    (e.is_recurring = false AND e.start_date = '2025-10-10')
    OR
    (e.is_recurring = true AND
     EXTRACT(DOW FROM '2025-10-10'::date) = ANY(e.recurrence_days_of_week))
  )
ORDER BY e.start_time;
```

### Taxa de cumprimento mensal por categoria

```sql
SELECT
  c.name,
  COUNT(ex.id) as total_executions,
  SUM(CASE WHEN ex.completed THEN 1 ELSE 0 END) as completed_count,
  ROUND(
    (SUM(CASE WHEN ex.completed THEN 1 ELSE 0 END)::numeric / COUNT(ex.id)) * 100,
    2
  ) as completion_rate
FROM categories c
JOIN events e ON c.id = e.category_id
JOIN event_executions ex ON e.id = ex.event_id
WHERE EXTRACT(YEAR FROM ex.execution_date) = 2025
  AND EXTRACT(MONTH FROM ex.execution_date) = 10
GROUP BY c.id, c.name
ORDER BY completion_rate DESC;
```

---

## 🚀 Migrations

### TypeORM Migration Command
```bash
npm run migration:generate -- -n CreateCalendarTables
npm run migration:run
```

### Prisma Migration Command
```bash
npx prisma migrate dev --name init
npx prisma generate
```

---

## 🔐 Security Considerations

1. **Row Level Security (RLS)**: Garantir que usuários só acessem seus próprios dados
2. **Soft Deletes**: Usar `is_active` ao invés de DELETE físico
3. **Audit Trail**: Manter `created_at` e `updated_at` em todas as tabelas
4. **Indexes**: Otimizar queries mais frequentes
5. **Foreign Keys**: Manter integridade referencial

---

## 📦 Next Steps

1. ✅ Definir schema completo
2. ⏳ Criar migrations
3. ⏳ Implementar entities no NestJS
4. ⏳ Criar repositories
5. ⏳ Implementar use cases
6. ⏳ Testar queries de performance
