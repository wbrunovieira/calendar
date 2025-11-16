# Testing Quick Reference

Comandos rápidos para trabalhar com testes no projeto Calendar.

---

## Setup Inicial (primeira vez)

```bash
# 1. Garantir que PostgreSQL está rodando
docker-compose up -d postgres

# 2. Criar banco de teste
bash scripts/setup-test-db.sh

# 3. Iniciar backend
docker-compose up -d calendar-core
```

---

## Comandos Diários

### Testes - calendar-core (NestJS)

```bash
# Rodar todos os testes
docker-compose exec calendar-core npm run test

# Watch mode (re-roda ao salvar)
docker-compose exec calendar-core npm run test:watch

# Com cobertura
docker-compose exec calendar-core npm run test:cov

# Interface visual
docker-compose exec calendar-core npm run test:ui

# Testes E2E
docker-compose exec calendar-core npm run test:e2e

# E2E em watch mode
docker-compose exec calendar-core npm run test:e2e:watch
```

### Testes - calendar-finances (Go)

```bash
# Todos os testes
docker-compose exec calendar-finances go test ./...

# Verbose
docker-compose exec calendar-finances go test -v ./...

# Pacote específico
docker-compose exec calendar-finances go test ./internal/domain/transaction/...

# Com cobertura
docker-compose exec calendar-finances go test -cover ./...

# Detectar race conditions
docker-compose exec calendar-finances go test -race ./...
```

---

## Banco de Dados de Teste

```bash
# Resetar banco (limpar tudo)
bash scripts/reset-test-db.sh

# Popular com dados de teste
bash scripts/seed-test-db.sh

# Recriar do zero
bash scripts/setup-test-db.sh

# Conectar via psql
psql -h localhost -p 5433 -U calendar -d calendar_test_db
```

---

## Criar Novos Testes

### Teste Unitário (calendar-core)

```typescript
// src/domains/[domain]/[layer]/[file].spec.ts
import { describe, it, expect, beforeEach } from 'vitest';
import { mock } from 'vitest-mock-extended';

describe('MyClass', () => {
  let instance: MyClass;
  let mockDep: MockProxy<Dependency>;

  beforeEach(() => {
    mockDep = mock<Dependency>();
    instance = new MyClass(mockDep);
  });

  it('should do something', () => {
    // Arrange
    const input = 'test';
    mockDep.method.mockReturnValue('result');

    // Act
    const result = instance.doSomething(input);

    // Assert
    expect(result).toBe('result');
    expect(mockDep.method).toHaveBeenCalledWith(input);
  });
});
```

### Teste Unitário (calendar-finances)

```go
// internal/[layer]/[package]/[file]_test.go
package mypackage_test

import (
    "testing"
    "github.com/brunovieira/calendar-finances/internal/test/helpers"
)

func TestMyFunction(t *testing.T) {
    // Arrange
    input := "test"
    expected := "result"

    // Act
    result := MyFunction(input)

    // Assert
    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
}
```

### Teste E2E (calendar-core)

```typescript
// test/[feature].e2e-spec.ts
import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import request from 'supertest';
import { Test } from '@nestjs/testing';
import { AppModule } from '../src/app.module';

describe('Feature API (e2e)', () => {
  let app: INestApplication;

  beforeAll(async () => {
    const moduleFixture = await Test.createTestingModule({
      imports: [AppModule],
    }).compile();

    app = moduleFixture.createNestApplication();
    await app.init();
  });

  it('should work', () => {
    return request(app.getHttpServer())
      .get('/endpoint')
      .expect(200);
  });

  afterAll(async () => {
    await app.close();
  });
});
```

---

## Usar Fixtures e Helpers

### calendar-core

```typescript
import {
  createEventFixture,
  createCalendarFixture,
  RRULE_DAILY,
  fixedTime,
} from '@/test/helpers/fixtures';

import { createMockRepository } from '@/test/helpers/mock-builders';

import { useFakeTimers } from '@/test/helpers/test-utils';

// No teste
const event = createEventFixture({ title: 'Custom Title' });
const mockRepo = createMockRepository<EventRepository>();
useFakeTimers(new Date('2024-11-16'));
```

### calendar-finances

```go
import "github.com/brunovieira/calendar-finances/internal/test/helpers"

// No teste
profile := helpers.CreateTestProfile("John Doe")
account := helpers.CreateTestBankAccount(profile.ID)
category := helpers.CreateExpenseCategory(profile.ID, "Food")
tx := helpers.CreateExpenseTransaction(
    profile.ID,
    account.ID,
    category.ID,
    100.00,
)
```

---

## Debugging

### Teste Específico (calendar-core)

```bash
# Rodar apenas um arquivo
docker-compose exec calendar-core npm run test -- src/domains/events/domain/entities/event.entity.spec.ts

# Rodar apenas testes que contenham "create"
docker-compose exec calendar-core npm run test -- --grep="create"
```

### Teste Específico (calendar-finances)

```bash
# Rodar apenas um arquivo
docker-compose exec calendar-finances go test ./internal/domain/transaction/transaction_test.go

# Rodar apenas uma função
docker-compose exec calendar-finances go test -run TestNewTransaction ./internal/domain/transaction/...
```

### Ver Output Completo

```bash
# calendar-core: adicionar -v
docker-compose exec calendar-core npm run test -- --reporter=verbose

# calendar-finances: já tem -v
docker-compose exec calendar-finances go test -v ./...
```

---

## Troubleshooting

### "vitest: not found"

```bash
# Reconstruir container
docker-compose down calendar-core
docker-compose up -d --build calendar-core
```

### "database does not exist"

```bash
# Recriar banco de teste
bash scripts/setup-test-db.sh
```

### "connection refused"

```bash
# Verificar se postgres está rodando
docker-compose ps
docker-compose up -d postgres
```

### Testes travando

```bash
# Limpar tudo e recomeçar
docker-compose down
docker-compose up -d
bash scripts/reset-test-db.sh
```

---

## Coverage Reports

### Ver cobertura no terminal

```bash
# calendar-core
docker-compose exec calendar-core npm run test:cov

# calendar-finances
docker-compose exec calendar-finances go test -cover ./...
```

### Coverage HTML Report (calendar-core)

```bash
docker-compose exec calendar-core npm run test:cov
# Abrir: services/calendar-core/coverage/index.html
```

### Coverage HTML Report (calendar-finances)

```bash
docker-compose exec calendar-finances go test -coverprofile=coverage.out ./...
docker-compose exec calendar-finances go tool cover -html=coverage.out -o coverage.html
# Abrir: services/calendar-finances/coverage.html
```

---

## CI/CD

### Ver status dos testes no GitHub

```
https://github.com/[username]/calendar/actions
```

### Rodar localmente como no CI

```bash
# Simular workflow do GitHub Actions
docker-compose down
docker-compose up -d postgres

# Esperar postgres iniciar
sleep 5

# calendar-core
cd services/calendar-core
npm ci
DATABASE_URL="postgresql://calendar:calendar123@localhost:5433/calendar_test_db" npx prisma migrate deploy
npm run test:cov

# calendar-finances
cd ../calendar-finances
go mod download
go test -v -race -coverprofile=coverage.out ./...
```

---

## Atalhos Úteis

```bash
# Aliases úteis (adicionar no ~/.bashrc ou ~/.zshrc)
alias tc='docker-compose exec calendar-core npm run test'
alias tcw='docker-compose exec calendar-core npm run test:watch'
alias tcc='docker-compose exec calendar-core npm run test:cov'
alias tce='docker-compose exec calendar-core npm run test:e2e'

alias tf='docker-compose exec calendar-finances go test ./...'
alias tfv='docker-compose exec calendar-finances go test -v ./...'
alias tfc='docker-compose exec calendar-finances go test -cover ./...'

alias tdb='bash scripts/reset-test-db.sh'
alias tdbs='bash scripts/seed-test-db.sh'
```

Depois de adicionar, recarregar:
```bash
source ~/.bashrc  # ou source ~/.zshrc
```

Usar:
```bash
tc     # rodar testes calendar-core
tcw    # watch mode
tf     # rodar testes calendar-finances
tdb    # resetar banco
```

---

## Dicas

1. **Use watch mode** durante desenvolvimento para feedback instantâneo
2. **Rode coverage** antes de fazer PR
3. **Limpe o banco** entre testes E2E complexos
4. **Use fixtures** para consistência nos dados de teste
5. **Mock APIs externas** (Google, Linear, Mercado Pago)
6. **Fixe datas** em testes para evitar flakiness
7. **Teste edge cases** (valores nulos, strings vazias, arrays vazios)
8. **Teste error paths** além dos happy paths

---

## Documentação Completa

- **Plano de Testes:** `docs/test-implementation-plan.md`
- **Resumo Fase 1:** `docs/phase-1-summary.md`
- **CLAUDE.md:** Seção "Testing"
