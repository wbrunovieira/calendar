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

## Fase 2: Backend - Sincronização Google Calendar
1. ✅ **CONCLUÍDO** - Criar API backend (NestJS)
   - ✅ Estrutura base do NestJS configurada
   - ✅ Dockerfile.dev criado
   - ✅ Estrutura DDD iniciada (`src/domains/google-calendar/domain/entities/`)
   - ✅ Entidade CalendarEvent criada
2. ❌ **PENDENTE** - Implementar autenticação OAuth2 Google
3. ❌ **PENDENTE** - Integrar Google Calendar API
4. ❌ **PENDENTE** - Criar endpoints para gerenciar duas contas
5. ❌ **PENDENTE** - Implementar sincronização de eventos

**Status da Fase 2**: 🟡 Iniciada (1/5 itens concluídos)

## Fase 3: Frontend - Interface
1. ❌ **PENDENTE** - Setup Next.js com TypeScript
2. ❌ **PENDENTE** - Criar layout base do calendário
3. ❌ **PENDENTE** - Implementar visualizações (dia/semana/mês)
4. ❌ **PENDENTE** - Criar sistema de login/autenticação
5. ❌ **PENDENTE** - Conectar com backend para sincronização

**Status da Fase 3**: ❌ Não iniciada (0/5 itens concluídos)

## Fase 4: Banco de Dados
1. ❌ **PENDENTE** - Definir schema do banco PostgreSQL
   - ✅ Container PostgreSQL configurado no docker-compose
   - ❌ Migrations não implementadas
   - ❌ Schema não definido
2. ❌ ~~**REMOVIDO** - Configurar Redis para cache e queues~~
   - Alternativas: NestJS cache-manager (memória) ou pg-boss (PostgreSQL)
3. ❌ **PENDENTE** - Implementar sistema de persistência
4. ❌ **PENDENTE** - Criar rotinas de backup automático

**Status da Fase 4**: 🟡 Parcialmente iniciada (infraestrutura PostgreSQL configurada)

## Fase 5: Worker Service
1. ❌ **PENDENTE** - Implementar worker em Go/Rust
2. ❌ **PENDENTE** - Sistema de filas com PostgreSQL (pg-boss ou BullMQ)
3. ❌ **PENDENTE** - Jobs de sincronização em massa
4. ❌ **PENDENTE** - Processamento batch e relatórios
5. ❌ **PENDENTE** - Preparação de dados para ML

**Status da Fase 5**: ❌ Não iniciada (0/5 itens concluídos)

## Fase 6: Serviço de IA
1. ❌ **PENDENTE** - Setup container Python com PyTorch/Llama
2. ❌ **PENDENTE** - Integração com Langchain/CrewAI
3. ❌ **PENDENTE** - Agentes especializados para análise
4. ❌ **PENDENTE** - Sistema de recomendações

**Status da Fase 6**: ❌ Não iniciada (0/4 itens concluídos)

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

### ✅ Implementado
- Docker Compose com calendar-core e PostgreSQL
- Estrutura base do NestJS com hot-reload
- Arquitetura DDD iniciada (domain/entities)
- Entidade CalendarEvent com atributos completos
- Configuração de ambiente (.env.example)
- Networks e volumes Docker
- PostgreSQL na porta 5433 (evita conflito com serviço local)

### 🔄 Em Andamento
- Fase 2: Backend Google Calendar (estrutura criada, integração pendente)

### ❌ Não Iniciado
- calendar-frontend (Next.js)
- calendar-ai (Python)
- calendar-worker (Go/Rust)
- Integrações: Google Calendar API, Linear API, Mercado Pago
- Schema e migrations do banco de dados
- Sistema de autenticação OAuth2
- Sistema de cache/filas (será PostgreSQL-based)
- Módulo financeiro completo

### 🗑️ Removido
- ~~Redis~~ - Desnecessário para projeto pessoal de usuário único. Alternativas mais econômicas: NestJS cache-manager (in-memory), pg-boss ou BullMQ com PostgreSQL adapter para filas.