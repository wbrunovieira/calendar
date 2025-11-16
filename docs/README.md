# Documentação de Testes - Calendar Application

Bem-vindo à documentação de testes do projeto Calendar! 🎯

---

## 📚 Índice de Documentos

### 🎯 Começando

1. **[Comandos Rápidos](testing-quick-reference.md)**
   - ⚡ Guia prático com comandos do dia a dia
   - Scripts de banco de dados
   - Como criar novos testes
   - Troubleshooting comum

2. **[Guia de Helpers](../services/calendar-core/src/test/README.md)**
   - 🔧 Como usar fixtures e mocks
   - Exemplos práticos
   - Boas práticas

### 📋 Planejamento e Progresso

3. **[Plano de Implementação](test-implementation-plan.md)**
   - 📊 Plano completo (16 semanas)
   - Progresso atual: Fase 1 ✅ Concluída
   - Próximas fases
   - Estratégias de teste por camada

4. **[Resumo da Fase 1](phase-1-summary.md)**
   - ✅ O que foi implementado
   - Estatísticas e métricas
   - Arquivos criados
   - Validações realizadas

### 📖 Referência Geral

5. **[CLAUDE.md](../CLAUDE.md#testing)**
   - Seção "Testing" completa
   - Padrões de teste
   - Organização de arquivos
   - Coverage goals

---

## 🚀 Quick Start

### Primeira vez rodando testes?

```bash
# 1. Criar banco de dados de teste
bash scripts/setup-test-db.sh

# 2. Iniciar serviços
docker-compose up -d

# 3. Rodar testes
docker-compose exec calendar-core npm run test
docker-compose exec calendar-finances go test ./...
```

### Comandos mais usados

```bash
# 🧪 Testes com watch mode (recomendado para desenvolvimento)
docker-compose exec calendar-core npm run test:watch

# 📊 Testes com cobertura
docker-compose exec calendar-core npm run test:cov

# 🎨 Interface visual (muito útil!)
docker-compose exec calendar-core npm run test:ui

# 🔄 Resetar banco de teste
bash scripts/reset-test-db.sh
```

---

## 📊 Status Atual

| Componente | Framework | Cobertura | Status |
|-----------|-----------|-----------|--------|
| **calendar-core** | Vitest | ~0% | ✅ Infraestrutura pronta |
| **calendar-finances** | Go Testing | ~11.5% | 🔄 Parcial |
| **Banco de Teste** | PostgreSQL | - | ✅ Configurado |
| **CI/CD** | GitHub Actions | - | ✅ Funcionando |
| **Helpers** | TS + Go | - | ✅ Criados |

**Fase Atual:** ✅ Fase 1 Concluída | 🔜 Fase 2 em Planejamento

---

## 🎯 Fase 1 Concluída (16/Nov/2024)

### ✅ Implementações

- [x] Migração Jest → Vitest completa
- [x] Banco de dados de teste (`calendar_test_db`)
- [x] Scripts de automação (setup/reset/seed)
- [x] GitHub Actions CI/CD
- [x] Helpers e fixtures (17+ funções)
- [x] Documentação completa (5 documentos)

### 📈 Métricas

- **Arquivos criados:** 14
- **Linhas de código:** ~1.200
- **Testes passando:** 1/1 (100%)
- **Tempo de implementação:** ~2 horas
- **Performance:** Vitest 10-20x mais rápido que Jest

---

## 🔜 Próximos Passos (Fase 2)

### Semana 3: RRuleHelper (Crítico)
- Testes de conversão RRULE
- Testes de timezone/DST
- Meta: 90%+ cobertura

### Semana 4: Events Domain
- Testes de entidades
- Testes de use cases complexos
- Meta: 80%+ cobertura

[Ver plano completo →](test-implementation-plan.md#7-cronograma-de-implementação)

---

## 🛠️ Estrutura de Arquivos

```
calendar/
├── docs/
│   ├── README.md                          # Este arquivo
│   ├── test-implementation-plan.md        # Plano completo
│   ├── phase-1-summary.md                 # Resumo Fase 1
│   └── testing-quick-reference.md         # Comandos rápidos
├── scripts/
│   ├── setup-test-db.sh                   # Setup banco
│   ├── reset-test-db.sh                   # Reset banco
│   └── seed-test-db.sh                    # Seed banco
├── .github/
│   └── workflows/
│       └── test.yml                       # CI/CD
├── services/
│   ├── calendar-core/
│   │   ├── vitest.config.ts               # Config Vitest
│   │   ├── vitest.config.e2e.ts           # Config E2E
│   │   ├── src/
│   │   │   └── test/
│   │   │       ├── setup.ts               # Setup global
│   │   │       ├── README.md              # Guia helpers
│   │   │       └── helpers/
│   │   │           ├── fixtures.ts        # Fixtures TS
│   │   │           ├── mock-builders.ts   # Mocks TS
│   │   │           └── test-utils.ts      # Utils TS
│   │   └── test/
│   │       └── setup-e2e.ts               # Setup E2E
│   └── calendar-finances/
│       └── internal/
│           └── test/
│               └── helpers/
│                   └── fixtures.go        # Fixtures Go
└── .env.test                              # Env testes
```

---

## 📖 Como Usar Esta Documentação

### Se você quer...

- **Rodar testes agora:** → [Comandos Rápidos](testing-quick-reference.md)
- **Criar um novo teste:** → [Guia de Helpers](../services/calendar-core/src/test/README.md)
- **Entender o plano geral:** → [Plano de Implementação](test-implementation-plan.md)
- **Ver o que foi feito:** → [Resumo Fase 1](phase-1-summary.md)
- **Referência completa:** → [CLAUDE.md](../CLAUDE.md#testing)

### Se você está...

- **Começando agora:** Leia [Comandos Rápidos](testing-quick-reference.md) + [Guia de Helpers](../services/calendar-core/src/test/README.md)
- **Implementando testes:** Use [Guia de Helpers](../services/calendar-core/src/test/README.md)
- **Planejando sprints:** Veja [Plano de Implementação](test-implementation-plan.md)
- **Revisando PR:** Confira [CLAUDE.md](../CLAUDE.md#testing) para padrões

---

## 🤝 Contribuindo

### Ao adicionar novos testes:

1. Use os helpers existentes ([Guia](../services/calendar-core/src/test/README.md))
2. Siga os padrões em [CLAUDE.md](../CLAUDE.md#testing)
3. Rode `npm run test:cov` para verificar cobertura
4. Documente casos de edge importantes

### Ao adicionar novos helpers:

1. Adicione em `src/test/helpers/`
2. Documente em `src/test/README.md`
3. Adicione exemplos de uso
4. Mantenha consistency com helpers existentes

---

## 📞 Suporte

- **Issues:** [GitHub Issues](https://github.com/seu-repo/calendar/issues)
- **Dúvidas sobre testes:** Veja [Troubleshooting](testing-quick-reference.md#troubleshooting)
- **Documentação Vitest:** [vitest.dev](https://vitest.dev/)
- **Documentação Go Testing:** [go.dev/doc/tutorial/add-a-test](https://go.dev/doc/tutorial/add-a-test)

---

## 🎓 Boas Práticas

1. **Sempre use fixtures** ao invés de criar objetos manualmente
2. **Fixe datas** com `fixedTime()` para testes determinísticos
3. **Limpe mocks** após cada teste com `afterEach()`
4. **Use watch mode** durante desenvolvimento
5. **Rode coverage** antes de abrir PR
6. **Mock APIs externas** (Google, Linear, Mercado Pago)
7. **Teste edge cases** e error paths

---

**Versão:** 1.0
**Última Atualização:** 16 de Novembro de 2024
**Status:** ✅ Fase 1 Concluída
