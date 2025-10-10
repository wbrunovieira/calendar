# Calendar App - Visão Geral

## Objetivo Principal
Aplicativo de gerenciamento de agenda pessoal e profissional com foco em **acompanhamento de hábitos**, **cumprimento de metas** e **análise de performance mensal**. O sistema permite categorizar eventos, marcar execução diária e gerar relatórios de progresso.

## Calendários

### 1. **WB Digital Solutions** (Profissional)
- **Email**: bruno@wbdigitalsolutions.com
- **Categorias**:
  - 💼 Reuniões
  - 📞 Prospecção
  - 💻 Programação
  - 💰 Financeiro
  - 📧 Emails/Comunicação
  - 📚 Estudo/Capacitação

### 2. **Pessoal** (Personal)
- **Email**: wbrunovieira77@gmail.com
- **Categorias**:
  - 🏃 Saúde (Academia, Jiu-Jitsu, Natação)
  - 📖 Lazer (Leitura, Hobbies)
  - 🧘 Bem-estar (Meditação, Yoga)
  - 👨‍👩‍👧 Família
  - 🎓 Educação/Cursos
  - 🏠 Casa/Tarefas

## Funcionalidades Principais

### 📅 Gestão de Eventos
- **Eventos Recorrentes**: Configuração de repetição (diária, semanal, mensal)
- **Eventos Únicos**: Compromissos pontuais
- **Categorização**: Cada evento pertence a uma categoria
- **Marcação de Execução**: Check/Uncheck diário para acompanhar o que foi feito
- **Cores por Categoria**: Identificação visual no calendário

### ✅ Tracking Diário
- Marcar eventos como **"Feito"** ou **"Não Feito"** cada dia
- Histórico de execução mantido para análise
- Visualização rápida do progresso diário

### 📊 Relatórios Mensais
- **Taxa de Cumprimento** por categoria
- **Gráficos de Performance**: Saúde, Trabalho, Lazer, etc.
- **Insights de IA**: Sugestões de melhoria baseadas em padrões
- **Comparação Mês a Mês**: Evolução ao longo do tempo

### 🎯 Categorias com Metas
- Definir **metas mensais** por categoria (ex: "Academia 3x/semana")
- Acompanhar progresso em tempo real
- Alertas de categorias negligenciadas

## Arquitetura de Containers

### Estrutura Docker
- **calendar-core** (NestJS): API principal, autenticação, CRUD de eventos, categorias, tracking
- **calendar-frontend** (Next.js): Interface web com calendário e dashboards
- **calendar-ai** (Python): Análise de padrões, geração de insights, recomendações
- **calendar-worker** (Go/Rust): Jobs de sincronização, backups, notificações
- **postgres**: Banco de dados principal

### Database Schema (Planejado)

#### Tabelas Principais:
- **users**: Usuários do sistema
- **calendars**: WB Digital Solutions, Pessoal
- **categories**: Categorias de eventos (com cores e ícones)
- **events**: Eventos (título, descrição, categoria, recorrência)
- **event_executions**: Registro de execução diária (feito/não feito)
- **monthly_goals**: Metas mensais por categoria
- **reports**: Relatórios mensais gerados

## Fluxo de Uso

### Configuração Inicial
1. Criar categorias para cada calendário
2. Definir eventos recorrentes (ex: "Academia" às 18h, 3x/semana)
3. Estabelecer metas mensais (opcional)

### Uso Diário
1. Visualizar agenda do dia
2. Marcar eventos como "Feito" conforme são executados
3. Receber lembretes de eventos pendentes

### Análise Mensal
1. Acessar dashboard de relatórios
2. Visualizar taxa de cumprimento por categoria
3. Identificar áreas de melhoria
4. Receber sugestões de IA

## Integração com Google Calendar

- **Sincronização Bidirecional**: Eventos criados no app aparecem no Google Calendar e vice-versa
- **Múltiplas Contas**: WB Digital e Pessoal
- **Mapeamento de Categorias**: Cores do Google Calendar sincronizadas com categorias locais

## Módulo Financeiro (Futuro)

### Integrações
- **Mercado Pago**: Transações e extratos
- **Nubank**: Import CSV/OFX

### Funcionalidades
- Dashboard financeiro separado
- Contas recorrentes aparecem no calendário
- Análise de gastos por categoria
- Alertas de vencimentos

## Stack Técnico

### Frontend
- Next.js 15.5 (App Router)
- React 19.1
- TypeScript
- Tailwind CSS 4

### Backend
- NestJS (TypeScript)
- PostgreSQL 15
- TypeORM / Prisma

### IA/ML (Futuro)
- Python
- Llama / GPT
- Langchain

### Infraestrutura
- Docker / Docker Compose
- Railway / Vercel (Deploy)

## Diferenciais

✅ **Foco em Hábitos**: Sistema pensado para tracking de rotinas recorrentes
✅ **Análise de Performance**: Relatórios visuais de progresso
✅ **Duas Personas**: Separação clara entre profissional e pessoal
✅ **IA Insights**: Recomendações baseadas em padrões de comportamento
✅ **Categorização Inteligente**: Organização visual por cores e ícones
✅ **Metas Mensuráveis**: Acompanhamento de objetivos com métricas claras

## Roadmap de Desenvolvimento

### Fase 1: Core (MVP)
- ✅ Setup Docker e containers
- ✅ Frontend básico com calendário
- ⏳ Backend: CRUD de eventos e categorias
- ⏳ Sistema de marcação de execução
- ⏳ Relatórios básicos

### Fase 2: Integração Google
- Google OAuth2
- Sincronização bidirecional
- Mapeamento de categorias

### Fase 3: Análise e IA
- Dashboard de relatórios avançados
- Insights de IA
- Recomendações personalizadas

### Fase 4: Módulo Financeiro
- Integração Mercado Pago / Nubank
- Dashboard financeiro
- Alertas de vencimentos
