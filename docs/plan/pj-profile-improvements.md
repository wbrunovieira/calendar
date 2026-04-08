# Melhorias para Perfil Jurídico (PJ)

**Data:** Abril 2026  
**Contexto:** O sistema hoje trata perfis PERSONAL e BUSINESS com a mesma estrutura e visão. Um perfil PJ tem necessidades distintas de controle, visualização e linguagem. Este documento descreve todas as melhorias planejadas.

---

## 1. Tipo de Pessoa Jurídica no Perfil

### Situação atual
O modelo `Profile` tem apenas `type: PERSONAL | BUSINESS`. Não há distinção entre MEI, ME, LTDA, SA etc. O sistema não sabe como apresentar a interface para um perfil PJ.

### O que mudar
Adicionar campo `legalEntityType` ao modelo `Profile`:

```
PERSONAL    → pessoa física (sem mudanças)
MEI         → Microempreendedor Individual
ME          → Microempresa (Simples Nacional)
EPP         → Empresa de Pequeno Porte
LTDA        → Sociedade Limitada
SA          → Sociedade Anônima
```

Adicionar também:
- `cnpj` (opcional, formatado)
- `companyName` (razão social, diferente do `name` que pode ser o nome fantasia)
- `openingDate` (data de abertura) — para cálculo de tempo de empresa
- `simplesNacional: boolean` — determina se DAS é calculado
- `taxRegime: SIMPLES | LUCRO_PRESUMIDO | LUCRO_REAL`

### Impacto na UI
Com `legalEntityType` preenchido, o sistema muda:
- Linguagem dos cartões (Faturamento em vez de Renda, Resultado em vez de Saldo)
- Habilita funcionalidades exclusivas PJ (DRE, DAS, Aportes, Ativos)
- No seletor de perfil, mostra o CNPJ ou razão social

### Cuidado com migração
- Campo `legalEntityType` deve ser **nullable** — perfis existentes não serão afetados
- Perfis BUSINESS existentes continuam funcionando sem preencher o novo campo
- A UI deve degradar graciosamente: se `legalEntityType` é null, mostra a interface atual

---

## 2. Centro de Custo / Cliente / Projeto como Dimensão Real

### Situação atual
O campo `cost_center` existe no modelo de transação mas não aparece em nenhum filtro, gráfico ou relatório. Usuário usa `description` e `notes` informalmente para identificar clientes.

### O que mudar
Transformar centro de custo em **primeira classe**:

- Criar entidade `CostCenter` com campos: `name`, `type (CLIENT | PROJECT | DEPARTMENT)`, `color`, `isActive`
- Vincular transações a um `costCenterId` (opcional)
- Filtros por cliente/projeto em: Lançamentos, Dashboard, Visão Mensal
- Gráficos novos no Dashboard PJ:
  - Receita por cliente (barras)
  - Concentração de receita (% por cliente — risco de dependência)
  - Resultado por projeto (receita - custos diretos)

### Impacto na migração
- `cost_center` hoje é texto livre na tabela — migrar para FK com `cost_center_id`
- Manter o campo texto como fallback enquanto migração não é feita
- Script de migração deve ser não-destrutivo: só adiciona coluna nova, não remove a antiga ainda

---

## 3. DRE Simplificada no Dashboard

### Situação atual
Dashboard mostra gráficos de categoria, tendência mensal e saldo acumulado. Não existe visão de resultado (lucro/prejuízo).

### O que adicionar
Nova aba ou seção no Dashboard (apenas para perfis PJ): **DRE do Mês**

```
(+) Receita Bruta              → soma de INCOME do período
(-) Deduções / Impostos        → categoria "Impostos" (DAS, ISS, etc.)
(=) Receita Líquida
(-) Custos Fixos               → categorias marcadas como fixo
(-) Custos Variáveis           → categorias marcadas como variável
(-) Pró-labore                 → categoria "Pró-labore"
(=) Resultado Operacional      → lucro ou prejuízo do mês
(-) Investimentos (Ads, etc.)  → categoria "Marketing / Anúncios"
(=) Resultado Final
```

Implementação: baseada em categorias, não requer mudança de modelo — só precisa que categorias sejam classificadas como `fixo | variavel | imposto | prolabore | marketing`.

Adicionar campo `classificationDRE` na entidade `Category`:
```
REVENUE         → receita bruta
TAX             → impostos e deduções  
FIXED_COST      → custo fixo operacional
VARIABLE_COST   → custo variável
PROLABORE       → retirada do sócio
MARKETING       → investimento em marketing/ads
FINANCIAL       → rendimentos, juros, IOF
ASSET           → compra de ativo
CAPITAL         → aporte de capital
```

---

## 4. Alerta e Cálculo de DAS (Simples Nacional)

### Situação atual
Nenhum controle fiscal. O DAS vence todo mês e não aparece no sistema.

### O que adicionar
Para perfis com `simplesNacional: true`:

- **Alerta na home** a partir do dia 15 de cada mês: "DAS de [mês] ainda não foi pago"
- **Estimativa automática**: baseada no faturamento do mês anterior, aplicar alíquota cadastrada no perfil
- Campo no perfil: `dasAliquota` (percentual — varia por atividade e faturamento acumulado)
- Ao confirmar pagamento do DAS, registrar como transação na categoria `TAX`

### Cuidado
Não calcular imposto automaticamente sem o usuário validar — apenas estimar e alertar. O cálculo real do Simples Nacional tem muitas variáveis.

---

## 5. Aportes do Sócio (Capital Investido na Empresa)

### Situação atual
Não existe. O usuário não tem como saber quanto já colocou de dinheiro próprio na empresa nem quanto a empresa lhe deve.

### O que adicionar
Nova entidade `CapitalContribution` (Aporte):

```
id
profileId
amount              → valor aportado
date                → data do aporte
description         → ex: "Capital inicial", "Cobertura deficit março"
type: CONTRIBUTION | WITHDRAWAL | LOAN
origin              → conta de origem (Nubank pessoal, etc.)
isReturned          → já foi devolvido?
returnedAmount      → quanto já foi devolvido
returnedAt
notes
```

**Visão nova: "Sócio x Empresa"** (ou dentro de Contas / Configurações):
- Total aportado pelo sócio
- Total devolvido (pró-labore + retiradas contam como devolução parcial? — a definir)
- Saldo devedor da empresa para o sócio
- Histórico de aportes

### Onde aparece na UI
- Nova aba em `/contas` ou seção dedicada em `/configuracoes`
- Valor "Empresa deve ao sócio: R$ X" visível no resumo financeiro do perfil PJ

### Impacto na migração
- Nova tabela `finance.capital_contributions` — apenas additive, sem risco

---

## 6. Ativos da Empresa (Patrimônio PJ)

### Situação atual
Não existe. Compra de computador, equipamento, móvel aparece apenas como EXPENSE e some.

### O que adicionar
Nova entidade `CompanyAsset` (Ativo):

```
id
profileId
name                → ex: "MacBook Pro M3"
category            → HARDWARE, SOFTWARE, FURNITURE, VEHICLE, REAL_ESTATE, OTHER
purchaseDate
purchaseAmount
currentValue        → valor atual (atualizado manualmente)
depreciationRate    → % ao ano (para cálculo automático)
depreciationMethod  → STRAIGHT_LINE | DECLINING_BALANCE
transactionId       → FK para a transação de compra (opcional)
notes
isActive            → se ainda está em uso
disposalDate        → quando foi descartado/vendido
disposalAmount
```

**Funcionalidades:**
- Ao registrar uma despesa, opção "Isso é um ativo da empresa" → cria o ativo vinculado
- Página `/ativos` (ou seção em `/contas`) listando todos os ativos com valor atual estimado
- Depreciação automática calculada para mostrar valor atual estimado
- **Patrimônio PJ** = soma dos ativos ativos ao valor atual

### Impacto na migração
- Nova tabela `finance.company_assets` — apenas additive

---

## 7. Controle de Anúncios Pagos e ROI

### Situação atual
Não existe. Um gasto em Meta Ads ou Google Ads entra como EXPENSE comum.

### O que adicionar
Estender o conceito de centro de custo com **rastreamento de campanha**:

Entidade `MarketingCampaign`:

```
id
profileId
name                → ex: "Google Ads - Revalida Apr 2026"
platform            → GOOGLE_ADS, META_ADS, INSTAGRAM, LINKEDIN, OTHER
startDate
endDate
budget              → orçamento planejado
totalSpent          → calculado das transações vinculadas
revenueAttributed   → receita atribuída a essa campanha (manual)
leads               → número de leads gerados (manual)
conversions         → número de conversões (manual)
costPerLead         → calculado: totalSpent / leads
costPerConversion   → calculado: totalSpent / conversions
roi                 → calculado: (revenueAttributed - totalSpent) / totalSpent * 100
notes
```

**Fluxo de uso:**
1. Usuário cria campanha com orçamento
2. Ao registrar gastos com anúncios, vincula à campanha
3. Quando contrato fecha, atribui a receita à campanha manualmente
4. Sistema calcula CPL, CPC, ROI

**Visão:**
- Nova seção "Marketing" no Dashboard PJ
- Cards: Total investido, Receita atribuída, ROI médio
- Tabela de campanhas com performance

### Impacto na migração
- Nova tabela `finance.marketing_campaigns` — apenas additive
- Nova coluna `campaign_id` na tabela de transações (nullable)

---

## 8. Reembolso Pessoal no Cartão PJ

### Situação atual
Usuário usa cartão PJ para despesas pessoais e registra o reembolso manualmente sem rastreamento.

### O que adicionar
Campo booleano `isPersonalReimbursement` na transação + campo `reimbursedAt`.

**Fluxo:**
- Ao criar transação de despesa no cartão PJ, marcar "É despesa pessoal (reembolsar)"
- A transação aparece com badge especial na lista
- Alerta na home: "X despesas pessoais aguardando reembolso — Total: R$ Y"
- Ao registrar o reembolso (INCOME), vincular às despesas originais

---

## 9. Reformulação da Linguagem para PJ

Quando o perfil tiver `legalEntityType` preenchido, renomear:

| Atual | PJ |
|---|---|
| Entradas | Faturamento / Receitas |
| Saídas | Despesas Operacionais |
| Disponível | Caixa Disponível |
| Patrimônio | Resultado do Período |
| Metas | Reservas e Objetivos |
| Lançamentos | Movimentações |

---

## 10. Reformulação das Metas para PJ

Metas pessoais (juntar para viagem, carro) não fazem sentido para PJ.

Para PJ, as metas relevantes são:
- **Reserva operacional** — equivalente a X meses de custo fixo
- **Fundo de imposto** — % do faturamento reservado para obrigações fiscais
- **Fundo de investimento** — capital para anúncios, equipamentos, contratações
- **Meta de faturamento** — receita mínima mensal / anual

Adicionar campo `goalType` na entidade `Goal`:
```
PERSONAL_SAVINGS    → uso pessoal (atual)
OPERATIONAL_RESERVE → reserva de caixa PJ
TAX_FUND            → fundo de imposto
INVESTMENT_FUND     → fundo de investimento em crescimento
REVENUE_TARGET      → meta de faturamento
```

---

## Ordem de Implementação Sugerida

### Fase 1 — Fundação ✅ CONCLUÍDA (Abril 2026)
1. ✅ Adicionar `legalEntityType`, `cnpj`, `companyName`, `simplesNacional`, `taxRegime`, `dasAliquota`, `openingDate` ao Profile (nullable — sem quebrar produção)
2. ✅ Adicionar `classificationDRE` à Category (nullable)
3. ✅ Nova tabela `finance.capital_contributions`
4. ✅ Nova tabela `finance.company_assets`
5. ✅ Atualizar UI do ProfileModal para mostrar campos PJ (seção amber, só quando BUSINESS)
6. ✅ Dropdown de `classificationDRE` na página de Categorias (só para BUSINESS)
7. ✅ Nova página `/aportes` — CRUD completo + cards de resumo ("Empresa deve ao sócio")
8. ✅ Nova página `/ativos` — CRUD completo + cards de patrimônio e depreciação
9. ✅ Nav links "Aportes" e "Ativos" visíveis apenas para perfis BUSINESS
10. ✅ Deploy em produção — `finances.wbdigitalsolutions.com`

### Fase 2 — Visibilidade
6. DRE simplificada no Dashboard (baseada nas classificações de categoria)
7. Visão "Sócio x Empresa" mais visível (saldo devedor no header/dashboard)
8. Alerta DAS na home para perfis com `simplesNacional: true`

### Fase 3 — Centro de Custo e Marketing
9. Entidade `CostCenter` como dimensão real
10. Migração do campo `cost_center` texto → FK (não-destrutiva)
11. Nova entidade `MarketingCampaign`
12. Fluxo de campanha + ROI

### Fase 4 — Refinamento
13. Campo `isPersonalReimbursement` nas transações
14. Reformulação das Metas com `goalType`
15. Renomear linguagem da UI condicionalmente ao `legalEntityType`

---

## Princípios de Migração

> **Regra de ouro: toda mudança de schema deve ser não-destrutiva.**

1. **Sempre nullable primeiro** — novos campos entram como nullable. Nunca adicionar coluna NOT NULL sem default em tabela com dados.
2. **Nunca remover coluna com dados** — deprecar logicamente antes de remover fisicamente (aguardar 2+ releases).
3. **FK nova = nullable** — `cost_center_id` entra nullable enquanto `cost_center` texto ainda existe.
4. **Testar migração em staging** — rodar `docker compose up` local com dump de produção antes de qualquer deploy.
5. **Migração reversível** — cada migration deve ter `down` funcional.
6. **Dados existentes não mudam** — perfis BUSINESS atuais continuam funcionando sem preencher `legalEntityType`. A UI mostra a interface atual por padrão.
