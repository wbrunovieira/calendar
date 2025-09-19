# Calendar App - Visão Geral

## Objetivo
Aplicativo de calendário com interface amigável para gerenciar e sincronizar agendas profissional e pessoal, integrando contas Google (bruno@wbdigitalsolutions.com e wbrunovieira77@gmail.com), com análise de dados e agentes IA especializados.

## Funcionalidades Principais
- Sincronização com Google Calendar (2 contas)
- Planejamento diário integrado (profissional/pessoal)
- Integração com Linear (gestão de tarefas)
- Auto-gestão de desenvolvimento via Linear API (tracking automático do próprio projeto)
- Módulo Financeiro com dashboard dedicado
- Armazenamento local de dados para análise
- Agentes IA para funções específicas
- Execução local com recursos cloud

## Módulo Financeiro
### Integrações
- **Mercado Pago**: API oficial para transações e extratos
- **Nubank Pessoal/Empresa**: Import de extratos CSV/OFX ou parser de emails

### Funcionalidades
- Dashboard financeiro separado (página dedicada no frontend)
- Calendário mostra apenas contas recorrentes (boletos, assinaturas)
- Análise de gastos com categorização automática
- Previsões e alertas de vencimentos
- Relatórios e insights via IA

## Arquitetura de Containers

### Estrutura Docker
- **calendar-core** (NestJS): API principal, autenticação, Google sync, Linear API
- **calendar-frontend** (Next.js): Interface web
- **calendar-ai** (Python): Serviços IA, agentes inteligentes
- **calendar-worker** (Go/Rust): Jobs pesados e processamento batch
- **postgres**: Banco de dados principal
- **redis**: Cache e message queue

### Jobs do Worker Service
- Sincronização em massa (histórico Google Calendar, imports CSV/ICS)
- Análise de padrões e geração de relatórios
- Processamento de dados para ML (embeddings, clustering)
- Operações recorrentes (backups, re-sync noturno, limpeza)
- Integrações pesadas (export bulk Linear, OCR, attachments)
- Notificações em batch (resumos, digests, alertas)

## Stack Disponível

### Frontend
- Next.js

### Backend
- NestJS
- Go
- Python
- Rust

### IA/ML
- Llama
- Hugging Face
- PyTorch

### Infraestrutura
- Docker
- N8N (cloud)
- Evolution API (cloud)
- Nextcloud
- Ansible
- Terraform

### Orquestração
- Agno
- Langchain
- LangGraph
- CrewAI