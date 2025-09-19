# Plano de Desenvolvimento

## Fase 1: Setup Inicial e Containers
1. Configurar Docker Compose com estrutura de containers:
   - calendar-core (NestJS)
   - calendar-frontend (Next.js)
   - calendar-ai (Python)
   - calendar-worker (Go/Rust)
   - PostgreSQL e Redis
2. Configurar networks e volumes Docker
3. Setup ambiente de desenvolvimento com hot-reload

## Fase 2: Backend - Sincronização Google Calendar
1. Criar API backend (NestJS)
2. Implementar autenticação OAuth2 Google
3. Integrar Google Calendar API
4. Criar endpoints para gerenciar duas contas
5. Implementar sincronização de eventos

## Fase 3: Frontend - Interface
1. Setup Next.js com TypeScript
2. Criar layout base do calendário
3. Implementar visualizações (dia/semana/mês)
4. Criar sistema de login/autenticação
5. Conectar com backend para sincronização

## Fase 4: Banco de Dados
1. Definir schema do banco PostgreSQL
2. Configurar Redis para cache e queues
3. Implementar sistema de persistência
4. Criar rotinas de backup automático

## Fase 5: Worker Service
1. Implementar worker em Go/Rust
2. Sistema de filas com Redis
3. Jobs de sincronização em massa
4. Processamento batch e relatórios
5. Preparação de dados para ML

## Fase 6: Serviço de IA
1. Setup container Python com PyTorch/Llama
2. Integração com Langchain/CrewAI
3. Agentes especializados para análise
4. Sistema de recomendações

## Fase 7: Módulo Financeiro
1. Dashboard financeiro (página separada no frontend)
2. Integração API Mercado Pago
3. Sistema de import Nubank (CSV/OFX)
4. Parser de emails de notificações bancárias
5. Eventos recorrentes no calendário (contas/boletos)
6. Categorização automática com ML
7. Análises e previsões financeiras

## Próximos Passos
- Integração Linear API para auto-gestão
- Analytics avançado e dashboards
- Automações com N8N
- Deploy com Terraform/Ansible