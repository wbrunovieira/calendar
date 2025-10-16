# Finance System Delivery Plan

## Escopo Principal
- [ ] Consolidar perfis pessoal e WB Digital Solutions com contas bancárias vinculadas
- [ ] Implementar lançamentos de receitas, despesas e transferências com atualização automática de saldos
- [ ] Oferecer dashboard web para registrar, visualizar e filtrar movimentos financeiros
- [ ] Preparar integrações futuras (importação de extratos, sincronização com calendário)

## Fases e Tarefas

### Fase 1 — Domínio e Dados
- [x] Definir modelo `Transaction` (tipos, status, parcelas, tags) alinhado às necessidades pessoal/empresa
- [x] Especificar categorias, centros de custo e eventuais splits entre contas
- [x] Criar migrations SQL para `finance.transactions`, `finance.categories`, `finance.transaction_splits`
- [x] Atualizar repositórios Go com interfaces e testes unitários para o novo domínio

### Fase 2 — API e Regras de Negócio
- [x] Implementar casos de uso CRUD (`CreateTransaction`, `ListTransactions`, `UpdateStatus`, `DeleteTransaction`)
- [x] Garantir validações de saldo, limites de cartão e recorrências
- [x] Expor endpoints REST (`/api/v1/transactions`) com filtros por perfil, conta, período e status
- [x] Cobrir com testes de integração/table-driven e configurar logs estruturados

### Fase 3 — Frontend Financeiro
- [x] Criar componentes `TransactionForm`, `CashflowSummary`, `TransactionsTable`
- [ ] Adicionar fluxo de cadastro/edição com seleção de conta, categoria, anexos e recorrência
- [x] Construir dashboard de fluxo de caixa (entrada vs saída, saldo por conta)
- [x] Implementar filtros avançados (perfil, tipo, período) e exportação CSV inicial (CSV pendente)

### Fase 4 — Integrações e Automação
- [ ] Sincronizar lançamentos recorrentes com eventos do `calendar-core`
- [ ] Permitir importação CSV/OFX com revisão e categorização assistida
- [ ] Configurar notificações para lançamentos não confirmados e alertas de orçamento
- [ ] Alinhar autenticação/autorização com a camada já usada pelo calendário

### Fase 5 — Governança e Métricas
- [ ] Criar trilha de auditoria (`finance.transaction_audit`) e versionamento de lançamentos
- [ ] Definir relatórios periódicos (DRE simplificada, cashflow, reservas) e geração em PDF/CSV
- [ ] Mapear KPIs (burn rate, margem, reservas, compromissos futuros) e expor via API/BI
- [ ] Monitorar serviço com métricas (Prometheus) e alertas operacionais

## Próximas Ações (Semana Atual)
- [ ] Produzir blueprint ER detalhado e validar com stakeholders
- [ ] Abrir issues por tarefa/fase no repositório com responsáveis e estimativas
- [ ] Prototipar UX do formulário de lançamento para feedback rápido

## Registro de Progresso
| Item | Responsável | Status | Última atualização |
| --- | --- | --- | --- |
| Fase 1 — Domínio e Dados | | ☑ | 2025-10-16 |
| Fase 2 — API e Regras | | ☑ | 2025-10-16 |
| Fase 3 — Frontend | | ☐ | |
| Fase 4 — Integrações | | ☐ | |
| Fase 5 — Governança | | ☐ | |
| Próximas ações | | ☐ | |

## Notas
- Reavaliar prioridades a cada encerramento de fase.
- Documentar decisões relevantes em `docs/` para manter alinhamento entre frontend e backend.
