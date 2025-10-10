# Plano de Desenvolvimento

## Fase 1: Setup Inicial e Containers
1. ✅ **CONCLUÍDO** - Configurar Docker Compose com estrutura de containers:
   - ✅ calendar-core (NestJS)
   - ❌ calendar-frontend (Next.js)
   - ❌ calendar-ai (Python)
   - ❌ calendar-worker (Go/Rust)
   - ✅ PostgreSQL
   - ❌ ~~Redis~~ (Removido - projeto pessoal não necessita)
2. ✅ **CONCLUÍDO** - Configurar networks e volumes Docker
3. ✅ **CONCLUÍDO** - Setup ambiente de desenvolvimento com hot-reload

**Status da Fase 1**: 🟡 Parcialmente concluída (3/3 itens principais, mas apenas calendar-core implementado)

**Decisão Arquitetural**: Redis foi removido por ser desnecessário para projeto pessoal de usuário único. Alternativas: cache em memória NestJS, filas com PostgreSQL (pg-boss), JWT stateless.

## Fase 2: Backend - Core API e Gerenciamento de Eventos

### 2.1 Banco de Dados e Migrations
1. ❌ **PENDENTE** - Definir schema completo do PostgreSQL
   - ❌ Tabela `users` (autenticação e perfil)
   - ❌ Tabela `calendars` (WB Digital Solutions, Pessoal)
   - ❌ Tabela `categories` (Academia, Reuniões, etc.)
   - ❌ Tabela `events` (eventos recorrentes e únicos)
   - ❌ Tabela `event_executions` (tracking diário de feito/não feito)
   - ❌ Tabela `event_reminders` (lembretes)
   - ❌ Tabela `monthly_goals` (metas mensais por categoria)
   - ❌ Tabela `reports` (relatórios mensais em cache)
2. ❌ **PENDENTE** - Criar migrations com TypeORM ou Prisma
3. ❌ **PENDENTE** - Criar indexes otimizados para queries frequentes
4. ❌ **PENDENTE** - Implementar seed data para categorias iniciais

### 2.2 Autenticação e Usuários
1. ❌ **PENDENTE** - Implementar JWT authentication
   - ❌ POST /auth/register - Criar conta
   - ❌ POST /auth/login - Login
   - ❌ POST /auth/logout - Logout
   - ❌ GET /auth/me - Dados do usuário logado
2. ❌ **PENDENTE** - Criar guards de autenticação NestJS
3. ❌ **PENDENTE** - Implementar domain/entities/User
4. ❌ **PENDENTE** - Implementar repositories e use cases

### 2.3 Gestão de Calendários
1. ❌ **PENDENTE** - Domain: Entidade Calendar e agregados
2. ❌ **PENDENTE** - Application: Use cases
   - ❌ CreateCalendar
   - ❌ UpdateCalendar
   - ❌ DeleteCalendar
   - ❌ ListUserCalendars
3. ❌ **PENDENTE** - Infrastructure: Controllers e repositórios
   - ❌ GET /calendars - Listar calendários
   - ❌ GET /calendars/:id - Detalhes
   - ❌ POST /calendars - Criar
   - ❌ PUT /calendars/:id - Atualizar
   - ❌ DELETE /calendars/:id - Remover

### 2.4 Gestão de Categorias
1. ❌ **PENDENTE** - Domain: Entidade Category
2. ❌ **PENDENTE** - Application: Use cases
   - ❌ CreateCategory
   - ❌ UpdateCategory
   - ❌ DeleteCategory
   - ❌ ListCalendarCategories
3. ❌ **PENDENTE** - Infrastructure: Controllers
   - ❌ GET /categories - Listar categorias
   - ❌ GET /categories/:id - Detalhes
   - ❌ POST /categories - Criar
   - ❌ PUT /categories/:id - Atualizar
   - ❌ DELETE /categories/:id - Remover

### 2.5 Gestão de Eventos
1. ❌ **PENDENTE** - Domain: Entidade Event com recorrência
2. ❌ **PENDENTE** - Implementar lógica de recorrência
   - ❌ Diária (daily)
   - ❌ Semanal (weekly) com dias da semana
   - ❌ Mensal (monthly) com dia do mês
   - ❌ Anual (yearly)
3. ❌ **PENDENTE** - Application: Use cases
   - ❌ CreateEvent
   - ❌ UpdateEvent
   - ❌ DeleteEvent
   - ❌ ListEvents (com filtros)
   - ❌ GetEventsByDate
   - ❌ GetEventsByDateRange
4. ❌ **PENDENTE** - Infrastructure: Controllers
   - ❌ GET /events - Listar com query params
   - ❌ GET /events/:id - Detalhes
   - ❌ POST /events - Criar
   - ❌ PUT /events/:id - Atualizar
   - ❌ DELETE /events/:id - Remover
   - ❌ GET /events/date/:date - Eventos do dia
   - ❌ GET /events/range - Período específico

### 2.6 Sistema de Execução (Tracking Diário)
1. ❌ **PENDENTE** - Domain: Entidade EventExecution
2. ❌ **PENDENTE** - Application: Use cases
   - ❌ MarkEventExecution (feito/não feito)
   - ❌ GetExecutionHistory
   - ❌ GetDailyExecutions
   - ❌ CalculateExecutionStats
3. ❌ **PENDENTE** - Infrastructure: Controllers
   - ❌ POST /events/:id/executions - Marcar execução
   - ❌ GET /events/:id/executions - Histórico
   - ❌ GET /executions/date/:date - Execuções do dia
   - ❌ GET /executions/stats - Estatísticas

### 2.7 Metas Mensais
1. ❌ **PENDENTE** - Domain: Entidade MonthlyGoal
2. ❌ **PENDENTE** - Application: Use cases
   - ❌ CreateMonthlyGoal
   - ❌ UpdateMonthlyGoal
   - ❌ DeleteMonthlyGoal
   - ❌ GetGoalProgress
3. ❌ **PENDENTE** - Infrastructure: Controllers
   - ❌ GET /goals - Listar metas
   - ❌ GET /goals/:id - Detalhes
   - ❌ POST /goals - Criar meta
   - ❌ PUT /goals/:id - Atualizar
   - ❌ DELETE /goals/:id - Remover
   - ❌ GET /goals/category/:id - Metas de categoria
   - ❌ GET /goals/month/:yearMonth - Metas do mês

### 2.8 Relatórios e Análises
1. ❌ **PENDENTE** - Domain: Entidade Report
2. ❌ **PENDENTE** - Application: Use cases
   - ❌ GenerateMonthlyReport
   - ❌ GetCategoryReport
   - ❌ CompareMonths
   - ❌ CalculateCompletionRates
3. ❌ **PENDENTE** - Infrastructure: Controllers
   - ❌ GET /reports/monthly/:yearMonth - Relatório mensal
   - ❌ GET /reports/category/:id - Por categoria
   - ❌ GET /reports/compare - Comparação entre meses
   - ❌ POST /reports/generate - Gerar relatório

### 2.9 Estatísticas e Analytics
1. ❌ **PENDENTE** - Implementar queries otimizadas
2. ❌ **PENDENTE** - Infrastructure: Controllers
   - ❌ GET /stats/daily/:date - Stats do dia
   - ❌ GET /stats/weekly - Stats da semana
   - ❌ GET /stats/monthly/:month - Stats do mês
   - ❌ GET /stats/category/:id - Stats por categoria
   - ❌ GET /stats/trends - Tendências

**Status da Fase 2**: 🟡 Planejamento completo, implementação pendente (0/9 módulos concluídos)

## Fase 3: Frontend - Interface

### 3.1 Setup e Configuração
1. ✅ **CONCLUÍDO** - Setup Next.js 15.5 com TypeScript
2. ✅ **CONCLUÍDO** - Configurar Tailwind CSS 4
3. ✅ **CONCLUÍDO** - Implementar tema visual (cores #350545 e #792990)
4. ✅ **CONCLUÍDO** - Criar componente Calendar base
5. ✅ **CONCLUÍDO** - Implementar visualizações (mês/semana/3 dias/dia)
6. ❌ **PENDENTE** - Configurar API client com axios/fetch
7. ❌ **PENDENTE** - Implementar gerenciamento de estado (Context API ou Zustand)

### 3.2 Autenticação
1. ❌ **PENDENTE** - Página de login
2. ❌ **PENDENTE** - Página de registro
3. ❌ **PENDENTE** - Gerenciamento de sessão JWT
4. ❌ **PENDENTE** - Protected routes
5. ❌ **PENDENTE** - Logout e refresh token

### 3.3 Calendário - Visualização de Eventos
1. ❌ **PENDENTE** - Integrar calendário com API de eventos
2. ❌ **PENDENTE** - Exibir eventos no calendário (com cores por categoria)
3. ❌ **PENDENTE** - Implementar tooltip com detalhes do evento
4. ❌ **PENDENTE** - Filtros por calendário (WB Digital / Pessoal)
5. ❌ **PENDENTE** - Filtros por categoria
6. ❌ **PENDENTE** - Indicadores visuais (eventos concluídos/pendentes)

### 3.4 Gestão de Eventos
1. ❌ **PENDENTE** - Modal de criação de evento
2. ❌ **PENDENTE** - Modal de edição de evento
3. ❌ **PENDENTE** - Formulário de recorrência (daily/weekly/monthly)
4. ❌ **PENDENTE** - Seletor de categoria
5. ❌ **PENDENTE** - Exclusão de evento com confirmação
6. ❌ **PENDENTE** - Drag and drop de eventos (opcional)

### 3.5 Execução Diária (Tracking)
1. ❌ **PENDENTE** - Checkbox para marcar evento como feito
2. ❌ **PENDENTE** - Interface de execução diária
3. ❌ **PENDENTE** - Campo de notas por execução
4. ❌ **PENDENTE** - Histórico de execuções do evento
5. ❌ **PENDENTE** - Indicadores visuais de progresso

### 3.6 Gestão de Categorias
1. ❌ **PENDENTE** - Página de gerenciamento de categorias
2. ❌ **PENDENTE** - Criar/editar categoria (nome, ícone, cor)
3. ❌ **PENDENTE** - Seletor de ícones (emojis)
4. ❌ **PENDENTE** - Color picker para categorias
5. ❌ **PENDENTE** - Listagem de categorias por calendário

### 3.7 Metas Mensais
1. ❌ **PENDENTE** - Página de metas mensais
2. ❌ **PENDENTE** - Formulário de criação de meta
3. ❌ **PENDENTE** - Progress bar visual de meta
4. ❌ **PENDENTE** - Listagem de metas do mês atual
5. ❌ **PENDENTE** - Alertas de metas em risco

### 3.8 Dashboard de Relatórios
1. ❌ **PENDENTE** - Página de relatório mensal
2. ❌ **PENDENTE** - Gráficos de taxa de cumprimento por categoria
3. ❌ **PENDENTE** - Comparação mês a mês
4. ❌ **PENDENTE** - Insights de IA (quando disponível)
5. ❌ **PENDENTE** - Exportar relatório (PDF/CSV)
6. ❌ **PENDENTE** - Visualização de tendências

### 3.9 UI/UX
1. ❌ **PENDENTE** - Modo responsivo (mobile/tablet)
2. ❌ **PENDENTE** - Loading states e skeletons
3. ❌ **PENDENTE** - Tratamento de erros
4. ❌ **PENDENTE** - Toasts de feedback
5. ❌ **PENDENTE** - Animações e transições

**Status da Fase 3**: 🟡 Interface básica criada, integração pendente (5/9 módulos iniciados)

## Fase 4: Sincronização Google Calendar

### 4.1 OAuth2 Google
1. ❌ **PENDENTE** - Configurar Google Cloud Console
2. ❌ **PENDENTE** - Implementar OAuth2 flow no backend
3. ❌ **PENDENTE** - Armazenar tokens de acesso (access_token, refresh_token)
4. ❌ **PENDENTE** - POST /auth/google/callback - Callback OAuth2
5. ❌ **PENDENTE** - Gerenciar múltiplas contas (WB Digital + Pessoal)

### 4.2 Integração Google Calendar API
1. ❌ **PENDENTE** - Listar calendários do Google
2. ❌ **PENDENTE** - Importar eventos do Google Calendar
3. ❌ **PENDENTE** - Exportar eventos para Google Calendar
4. ❌ **PENDENTE** - Sincronização bidirecional
5. ❌ **PENDENTE** - POST /sync/google/import - Import manual
6. ❌ **PENDENTE** - POST /sync/google/export - Export manual
7. ❌ **PENDENTE** - GET /sync/google/status - Status da sincronização

### 4.3 Mapeamento e Sincronização
1. ❌ **PENDENTE** - Mapear categorias locais para cores do Google
2. ❌ **PENDENTE** - Salvar google_calendar_id nos calendars
3. ❌ **PENDENTE** - Salvar google_event_id nos events
4. ❌ **PENDENTE** - Implementar webhooks do Google Calendar
5. ❌ **PENDENTE** - Sincronização automática em background

**Status da Fase 4**: ❌ Não iniciada (0/3 módulos concluídos)

## Fase 5: Worker Service (Opcional - Fase Avançada)

### 5.1 Setup Worker
1. ❌ **PENDENTE** - Escolher tecnologia (Go/Rust/Node.js)
2. ❌ **PENDENTE** - Setup Docker container
3. ❌ **PENDENTE** - Configurar conexão PostgreSQL

### 5.2 Sistema de Filas
1. ❌ **PENDENTE** - Implementar pg-boss (PostgreSQL-based queues)
2. ❌ **PENDENTE** - Criar jobs de sincronização Google Calendar
3. ❌ **PENDENTE** - Jobs de geração de relatórios mensais
4. ❌ **PENDENTE** - Jobs de limpeza e manutenção

### 5.3 Processamento Batch
1. ❌ **PENDENTE** - Import em massa de eventos (CSV/ICS)
2. ❌ **PENDENTE** - Cálculo de estatísticas agregadas
3. ❌ **PENDENTE** - Geração de backups automáticos
4. ❌ **PENDENTE** - Preparação de dados para ML

**Status da Fase 5**: ❌ Não iniciada - Prioridade baixa (0/3 módulos concluídos)

## Fase 6: Serviço de IA (Fase Avançada)

### 6.1 Setup AI Service
1. ❌ **PENDENTE** - Setup container Python
2. ❌ **PENDENTE** - Instalar PyTorch/Transformers
3. ❌ **PENDENTE** - Configurar Llama ou GPT API
4. ❌ **PENDENTE** - Integrar Langchain

### 6.2 Análise de Dados
1. ❌ **PENDENTE** - Análise de padrões de comportamento
2. ❌ **PENDENTE** - Identificação de tendências
3. ❌ **PENDENTE** - Detecção de anomalias (categorias negligenciadas)
4. ❌ **PENDENTE** - Previsão de cumprimento de metas

### 6.3 Geração de Insights
1. ❌ **PENDENTE** - Gerar insights textuais (mensais)
2. ❌ **PENDENTE** - Recomendações personalizadas
3. ❌ **PENDENTE** - Comparação com histórico
4. ❌ **PENDENTE** - Sugestões de otimização de agenda

### 6.4 Agentes Especializados (CrewAI)
1. ❌ **PENDENTE** - Agente de análise de produtividade
2. ❌ **PENDENTE** - Agente de saúde e bem-estar
3. ❌ **PENDENTE** - Agente financeiro (fase 7)
4. ❌ **PENDENTE** - Coordenação entre agentes

**Status da Fase 6**: ❌ Não iniciada - Prioridade baixa (0/4 módulos concluídos)

## Fase 7: Módulo Financeiro
1. ❌ **PENDENTE** - Dashboard financeiro (página separada no frontend)
2. ❌ **PENDENTE** - Integração API Mercado Pago
3. ❌ **PENDENTE** - Sistema de import Nubank (CSV/OFX)
4. ❌ **PENDENTE** - Parser de emails de notificações bancárias
5. ❌ **PENDENTE** - Eventos recorrentes no calendário (contas/boletos)
6. ❌ **PENDENTE** - Categorização automática com ML
7. ❌ **PENDENTE** - Análises e previsões financeiras

**Status da Fase 7**: ❌ Não iniciada (0/7 itens concluídos)

## Próximos Passos
- ❌ Integração Linear API para auto-gestão
- ❌ Analytics avançado e dashboards
- ❌ Automações com N8N
- ❌ Deploy com Terraform/Ansible

---

## Resumo Geral do Projeto

### ✅ Implementado (Fase 1 e 3 - Parcial)
**Infraestrutura:**
- Docker Compose com calendar-core e PostgreSQL
- PostgreSQL na porta 5433 (evita conflito com serviço local)
- Networks e volumes Docker
- Estrutura base do NestJS com hot-reload
- Arquitetura DDD iniciada (domain/entities)
- Configuração de ambiente (.env.example)

**Frontend:**
- Next.js 15.5 com TypeScript e Tailwind CSS 4
- Componente Calendar com 4 visualizações (mês/semana/3 dias/dia)
- Tema visual customizado (#350545, #792990)
- Interface responsiva básica
- Navegação entre períodos

**Documentação:**
- PROJECT_OVERVIEW.md - Visão geral e objetivos
- DATABASE_SCHEMA.md - Schema completo PostgreSQL com 8 tabelas
- BACKEND_ROUTES.md - Especificação completa de rotas API
- PLAN.md - Roadmap detalhado de implementação

### 🔄 Próximas Prioridades (MVP)

**Alta Prioridade:**
1. **Fase 2.1** - Database e Migrations
   - Implementar schema PostgreSQL completo
   - Criar migrations (TypeORM ou Prisma)
   - Seed data para categorias iniciais

2. **Fase 2.2** - Autenticação
   - JWT authentication
   - Rotas de registro e login
   - Guards NestJS

3. **Fase 2.3 a 2.6** - Backend Core
   - CRUD de Calendários
   - CRUD de Categorias
   - CRUD de Eventos (com recorrência)
   - Sistema de Execução Diária

4. **Fase 3.2 a 3.5** - Frontend Integração
   - Páginas de login/registro
   - Integração com API de eventos
   - Interface de tracking diário
   - Gestão de categorias

**Média Prioridade:**
5. **Fase 2.7** - Metas Mensais
6. **Fase 2.8** - Relatórios
7. **Fase 3.6 a 3.8** - UI de Metas e Relatórios

**Baixa Prioridade (Pós-MVP):**
8. **Fase 4** - Google Calendar Sync
9. **Fase 5** - Worker Service (opcional)
10. **Fase 6** - IA e Insights
11. **Fase 7** - Módulo Financeiro

### ❌ Não Iniciado (Core)
- Schema e migrations do banco de dados
- Sistema de autenticação completo
- Todas as rotas de backend (9 módulos)
- Integração frontend-backend
- Sistema de execução diária
- Metas mensais
- Relatórios e analytics

### ❌ Não Iniciado (Avançado)
- Google Calendar OAuth2 e sincronização
- calendar-ai (Python)
- calendar-worker (Go/Rust)
- Integrações: Linear API, Mercado Pago
- Módulo financeiro completo
- Sistema de IA com insights

### 🗑️ Removido
- ~~Redis~~ - Desnecessário para projeto pessoal. Alternativas: NestJS cache-manager (in-memory), pg-boss com PostgreSQL para filas.

### 📊 Progresso por Fase
- **Fase 1** (Setup): 🟡 60% (3/5 containers)
- **Fase 2** (Backend Core): 🔴 0% (0/9 módulos)
- **Fase 3** (Frontend): 🟡 15% (1.5/9 módulos)
- **Fase 4** (Google Sync): 🔴 0% (0/3 módulos)
- **Fase 5** (Worker): 🔴 0% (0/3 módulos)
- **Fase 6** (IA): 🔴 0% (0/4 módulos)
- **Fase 7** (Financeiro): 🔴 0% (0/7 módulos)