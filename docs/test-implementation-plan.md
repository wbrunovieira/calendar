# Plano de Implementação de Testes - Calendar Application

**Data de Criação:** Novembro 2024
**Última Atualização:** 16 de Novembro de 2024
**Status:** ✅ **Fase 1 Concluída** | 🚧 Fase 2 em Planejamento
**Framework:** Vitest para NestJS (calendar-core) | Go Testing para calendar-finances

---

## 📊 Progresso Geral

| Fase | Status | Progresso | Duração | Conclusão |
|------|--------|-----------|---------|-----------|
| **Fase 1** - Setup e Fundação | ✅ **Concluída** | 100% (13/13 tarefas) | ~2 horas | 16/Nov/2024 |
| **Fase 2** - Testes Críticos | 🔜 Próxima | 0% (0/4 semanas) | - | - |
| **Fase 3** - CRUD e Repositories | ⏸️ Pendente | 0% (0/4 semanas) | - | - |
| **Fase 4** - Integração e E2E | ⏸️ Pendente | 0% (0/4 semanas) | - | - |
| **Fase 5** - Refinamento | ⏸️ Pendente | 0% (0/2 semanas) | - | - |

**Progresso Total:** 12.5% (2 de 16 semanas) 🎯

---

## 🎯 Conquistas da Fase 1

✅ Vitest configurado e funcionando (1 teste passando)
✅ Banco de dados de teste criado (6 migrations aplicadas)
✅ Scripts de automação prontos (setup/reset/seed)
✅ GitHub Actions CI/CD configurado
✅ Helpers e fixtures criados (17+ funções utilitárias)
✅ Documentação completa (4 documentos)

**Ver detalhes:** `docs/phase-1-summary.md`

---

## 1. Estado Atual (Atualizado - 16/Nov/2024)

### calendar-core (NestJS)
- **Cobertura Atual:** ~0% (infraestrutura pronta, testes a serem escritos)
- **Framework Atual:** ✅ **Vitest** (migração concluída)
- **Arquivos de Teste Existentes:** 1
  - `src/app.controller.spec.ts` - Teste básico ✅ passando
- **Infraestrutura de Testes:**
  - ✅ Vitest configurado (`vitest.config.ts`, `vitest.config.e2e.ts`)
  - ✅ Banco de teste: `calendar_test_db` (6 migrations aplicadas)
  - ✅ Setup global: `src/test/setup.ts`
  - ✅ Setup E2E: `test/setup-e2e.ts`
  - ✅ Helpers: 3 arquivos (~400 linhas)
    - `fixtures.ts` - 10+ fixtures reutilizáveis
    - `mock-builders.ts` - Builders para mocks
    - `test-utils.ts` - Utilitários (fake timers, spies, etc.)

### calendar-finances (Go)
- **Cobertura Estimada:** ~11.5%
- **Framework:** Go Testing nativo + Custom SQLMock
- **Arquivos de Teste Existentes:** 7
  - Domain: `transaction_test.go`, `split_test.go`, `category_test.go`
  - Use Cases: `transaction_usecases_test.go`
  - Repositories: `transaction_repository_test.go`, `category_repository_test.go`, `budget_target_repository_test.go`
  - Infraestrutura: `internal/test/sqlmock/` (364 linhas)
- **Infraestrutura de Testes:**
  - ✅ Helpers: `internal/test/helpers/fixtures.go` (~200 linhas)
    - 15+ funções fixture para Profile, BankAccount, Category, Transaction
    - Helpers para datas, UUIDs, e conversões

### Banco de Dados de Teste
- ✅ **Nome:** `calendar_test_db`
- ✅ **URL:** `postgresql://calendar:calendar123@localhost:5433/calendar_test_db`
- ✅ **Scripts:**
  - `scripts/setup-test-db.sh` - Criar e migrar
  - `scripts/reset-test-db.sh` - Limpar
  - `scripts/seed-test-db.sh` - Popular
- ✅ **Isolamento:** Completamente separado do banco de desenvolvimento

---

## 2. Objetivos

### Metas de Cobertura
- **Fase 1 (1-2 meses):** 60% de cobertura em ambos os serviços
- **Fase 2 (3-4 meses):** 80% de cobertura em ambos os serviços
- **Fase 3 (5-6 meses):** 90%+ em código crítico

### Tipos de Teste
1. **Testes Unitários:** Domínios, casos de uso, utilitários
2. **Testes de Integração:** Repositórios, banco de dados
3. **Testes E2E:** Fluxos completos da API

---

## 3. Migração Jest → Vitest (calendar-core)

### 3.1. Justificativa
- **Performance:** Vitest é 10-20x mais rápido que Jest
- **ESM Nativo:** Melhor suporte a módulos ES
- **Compatibilidade:** API compatível com Jest (migração suave)
- **Dev Experience:** Hot Module Replacement (HMR) em testes
- **TypeScript:** Melhor integração nativa

### 3.2. Etapas de Migração

#### Passo 1: Instalação de Dependências
```bash
cd services/calendar-core
npm install -D vitest @vitest/ui @vitest/coverage-v8
npm install -D vite unplugin-swc @swc/core
npm uninstall jest ts-jest @types/jest
```

#### Passo 2: Configuração do Vitest
Criar `vitest.config.ts`:
```typescript
import { defineConfig } from 'vitest/config';
import swc from 'unplugin-swc';
import { resolve } from 'path';

export default defineConfig({
  test: {
    globals: true,
    environment: 'node',
    root: './',
    include: ['src/**/*.spec.ts'],
    exclude: ['node_modules', 'dist', 'test'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: [
        'node_modules/',
        'dist/',
        'test/',
        '**/*.dto.ts',
        '**/*.module.ts',
        'src/main.ts',
      ],
    },
  },
  plugins: [swc.vite()],
  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
      '@domains': resolve(__dirname, './src/domains'),
      '@common': resolve(__dirname, './src/common'),
    },
  },
});
```

#### Passo 3: Configuração E2E
Criar `vitest.config.e2e.ts`:
```typescript
import { defineConfig } from 'vitest/config';
import swc from 'unplugin-swc';

export default defineConfig({
  test: {
    globals: true,
    environment: 'node',
    root: './',
    include: ['test/**/*.e2e-spec.ts'],
    testTimeout: 30000,
    hookTimeout: 30000,
  },
  plugins: [swc.vite()],
});
```

#### Passo 4: Atualizar Scripts no package.json
```json
{
  "scripts": {
    "test": "vitest run",
    "test:watch": "vitest",
    "test:ui": "vitest --ui",
    "test:cov": "vitest run --coverage",
    "test:e2e": "vitest run --config vitest.config.e2e.ts",
    "test:e2e:watch": "vitest --config vitest.config.e2e.ts"
  }
}
```

#### Passo 5: Setup Global de Testes
Criar `src/test/setup.ts`:
```typescript
import { beforeAll, afterAll, afterEach } from 'vitest';

// Mock de variáveis de ambiente para testes
beforeAll(() => {
  process.env.DATABASE_URL = 'postgresql://test:test@localhost:5433/test_db';
  process.env.JWT_SECRET = 'test-secret';
  process.env.NODE_ENV = 'test';
});

afterEach(() => {
  // Limpar mocks após cada teste
  vi.clearAllMocks();
});

afterAll(() => {
  // Cleanup global
});
```

---

## 4. Estratégia de Testes - calendar-core

### 4.1. Priorização (Matriz de Risco x Complexidade)

| Prioridade | Componente | Complexidade | Risco | Estimativa |
|-----------|-----------|--------------|-------|-----------|
| **CRÍTICA** | RRuleHelper | Alta | Alto | 2 semanas |
| **CRÍTICA** | ListEventsUseCase | Alta | Alto | 1 semana |
| **CRÍTICA** | GetEventsStatsUseCase | Alta | Médio | 1 semana |
| **ALTA** | CreateEventUseCase | Média | Alto | 3 dias |
| **ALTA** | ToggleEventExecutionUseCase | Média | Médio | 2 dias |
| **MÉDIA** | CRUD Use Cases (4 domínios) | Baixa | Baixo | 1 semana |
| **MÉDIA** | Repositories (5 repos) | Média | Médio | 1 semana |
| **BAIXA** | Controllers | Baixa | Baixo | 3 dias |
| **BAIXA** | Entities | Baixa | Baixo | 2 dias |

### 4.2. Testes Unitários

#### 4.2.1. Domain Layer - Entities
**Arquivo:** `src/domains/events/domain/entities/event.entity.spec.ts`

**Cenários de Teste:**
```typescript
describe('Event Entity', () => {
  describe('create', () => {
    it('should create event with required fields');
    it('should generate UUID automatically');
    it('should set default status to CONFIRMED');
    it('should validate recurrence rule format');
    it('should fail without title');
    it('should fail without calendarId');
  });

  describe('recurrence', () => {
    it('should accept valid RRULE string');
    it('should reject invalid RRULE format');
    it('should handle null recurrence (non-recurring)');
  });

  describe('status', () => {
    it('should accept CONFIRMED status');
    it('should accept CANCELLED status');
    it('should accept TENTATIVE status');
    it('should reject invalid status');
  });
});
```

**Mock Strategy:**
```typescript
const mockEventData = {
  title: 'Test Event',
  calendarId: 'calendar-uuid',
  startDate: '2024-11-16',
  startTime: '10:00',
  duration: 60,
};
```

#### 4.2.2. Domain Layer - RRuleHelper (CRÍTICO)
**Arquivo:** `src/domains/events/domain/utils/rrule-helper.spec.ts`

**Cenários de Teste:**
```typescript
describe('RRuleHelper', () => {
  describe('toString', () => {
    it('should convert daily recurrence to RRULE');
    it('should convert weekly recurrence with byweekday');
    it('should convert monthly recurrence with bymonthday');
    it('should convert yearly recurrence');
    it('should handle interval parameter');
    it('should handle until date');
    it('should handle count parameter');
  });

  describe('parse', () => {
    it('should parse RRULE string to options');
    it('should extract frequency correctly');
    it('should extract interval');
    it('should extract byweekday array');
    it('should extract until date');
    it('should handle malformed RRULE gracefully');
  });

  describe('generateOccurrences', () => {
    describe('DAILY', () => {
      it('should generate daily occurrences');
      it('should respect interval (every 2 days)');
      it('should stop at until date');
      it('should respect count limit');
    });

    describe('WEEKLY', () => {
      it('should generate weekly occurrences');
      it('should filter by weekdays (MO, WE, FR)');
      it('should handle interval (every 2 weeks)');
      it('should cross month boundaries correctly');
    });

    describe('MONTHLY', () => {
      it('should generate monthly occurrences');
      it('should handle bymonthday (15th of each month)');
      it('should handle months with different lengths');
      it('should skip invalid dates (e.g., Feb 30)');
    });

    describe('YEARLY', () => {
      it('should generate yearly occurrences');
      it('should handle leap years');
    });

    describe('timezone handling', () => {
      it('should convert UTC to America/Sao_Paulo');
      it('should handle daylight saving time transitions');
      it('should maintain time consistency across DST');
    });

    describe('edge cases', () => {
      it('should handle range before event start');
      it('should handle range with no occurrences');
      it('should handle infinite recurrence (no until/count)');
      it('should limit occurrences to 1000 for safety');
    });
  });
});
```

**Mock Strategy:**
```typescript
// Fixar data/hora para testes determinísticos
vi.useFakeTimers();
vi.setSystemTime(new Date('2024-11-16T10:00:00-03:00'));

// Testar em diferentes timezones
process.env.TZ = 'America/Sao_Paulo';
```

#### 4.2.3. Application Layer - Use Cases
**Arquivo:** `src/domains/events/application/use-cases/list-events.use-case.spec.ts`

**Cenários de Teste:**
```typescript
describe('ListEventsUseCase', () => {
  let useCase: ListEventsUseCase;
  let mockEventRepository: MockProxy<EventRepository>;
  let mockRRuleHelper: MockProxy<typeof RRuleHelper>;

  beforeEach(() => {
    mockEventRepository = mock<EventRepository>();
    useCase = new ListEventsUseCase(mockEventRepository);
  });

  describe('execute', () => {
    it('should list events without filters');
    it('should filter by calendarId');
    it('should filter by categoryId');
    it('should search by title');
    it('should filter by date range');

    describe('recurrence expansion', () => {
      it('should expand daily recurring events');
      it('should expand weekly recurring events');
      it('should include master event if in range');
      it('should exclude exceptions from occurrences');
      it('should apply overrides to occurrences');
      it('should handle events with no recurrence');
      it('should not expand events outside date range');
    });

    describe('legacy format conversion', () => {
      it('should convert RRULE to legacy format');
      it('should include recurrenceLegacy field');
    });

    describe('error handling', () => {
      it('should handle repository errors gracefully');
      it('should return empty array when no events found');
    });
  });
});
```

**Mock Strategy:**
```typescript
// Mock do repositório
mockEventRepository.findAll.mockResolvedValue([
  {
    id: 'event-1',
    title: 'Daily Standup',
    recurrenceRule: 'FREQ=DAILY;INTERVAL=1',
    startDate: '2024-11-01',
    // ...
  }
]);

// Mock do RRuleHelper
vi.spyOn(RRuleHelper, 'generateOccurrences').mockReturnValue([
  new Date('2024-11-01'),
  new Date('2024-11-02'),
  new Date('2024-11-03'),
]);
```

#### 4.2.4. Infrastructure Layer - Repositories
**Arquivo:** `src/domains/events/infrastructure/repositories/event.repository.spec.ts`

**Cenários de Teste:**
```typescript
describe('EventRepository', () => {
  let repository: EventRepository;
  let prisma: DeepMockProxy<PrismaClient>;

  beforeEach(() => {
    prisma = mockDeep<PrismaClient>();
    repository = new EventRepository();
    // Inject mocked Prisma
    (repository as any).prisma = prisma;
  });

  describe('create', () => {
    it('should create event with Prisma');
    it('should return domain entity');
    it('should handle Prisma errors');
  });

  describe('findAll', () => {
    it('should query events with filters');
    it('should include relations (category, categoryType)');
    it('should map Prisma models to domain entities');
  });

  describe('findById', () => {
    it('should find event by id');
    it('should return null if not found');
  });

  describe('update', () => {
    it('should update event fields');
    it('should handle partial updates');
  });

  describe('delete', () => {
    it('should delete event');
    it('should return boolean indicating success');
  });
});
```

**Mock Strategy:**
```typescript
import { mockDeep, DeepMockProxy } from 'vitest-mock-extended';

// Mock Prisma Client
prisma.event.create.mockResolvedValue({
  id: 'uuid',
  title: 'Test',
  // ... Prisma model
});
```

#### 4.2.5. Infrastructure Layer - Controllers
**Arquivo:** `src/domains/events/infrastructure/controllers/events.controller.spec.ts`

**Cenários de Teste:**
```typescript
describe('EventsController', () => {
  let controller: EventsController;
  let mockCreateUseCase: MockProxy<CreateEventUseCase>;
  let mockListUseCase: MockProxy<ListEventsUseCase>;

  beforeEach(() => {
    mockCreateUseCase = mock<CreateEventUseCase>();
    mockListUseCase = mock<ListEventsUseCase>();
    controller = new EventsController(
      mockCreateUseCase,
      mockListUseCase,
      // ... other use cases
    );
  });

  describe('POST /events', () => {
    it('should call CreateEventUseCase');
    it('should return 201 status');
    it('should validate DTO');
    it('should return error on validation failure');
  });

  describe('GET /events', () => {
    it('should call ListEventsUseCase with filters');
    it('should return 200 status');
    it('should handle empty results');
  });

  describe('PUT /events/:id', () => {
    it('should update event');
    it('should return 200 status');
  });

  describe('DELETE /events/:id', () => {
    it('should delete event');
    it('should return 204 status');
  });

  describe('POST /events/executions/toggle', () => {
    it('should toggle execution status');
    it('should mark as completed');
    it('should mark as incomplete');
  });

  describe('GET /events/stats', () => {
    it('should return stats grouped by day');
    it('should return stats grouped by category');
    it('should handle different groupBy options');
  });
});
```

### 4.3. Testes de Integração

#### 4.3.1. Repository Integration Tests
**Arquivo:** `test/integration/event.repository.integration.spec.ts`

**Setup:**
```typescript
import { PrismaClient } from '@prisma/client';

describe('EventRepository Integration', () => {
  let prisma: PrismaClient;
  let repository: EventRepository;

  beforeAll(async () => {
    // Conectar ao banco de teste
    prisma = new PrismaClient({
      datasources: {
        db: { url: process.env.DATABASE_URL },
      },
    });
    repository = new EventRepository();
  });

  beforeEach(async () => {
    // Limpar dados de teste
    await prisma.event.deleteMany();
    await prisma.calendar.deleteMany();
  });

  afterAll(async () => {
    await prisma.$disconnect();
  });

  it('should create and retrieve event from database');
  it('should update event in database');
  it('should delete event from database');
  it('should query with complex filters');
});
```

**Requisitos:**
- Banco de dados de teste separado
- Migrations executadas antes dos testes
- Cleanup automático entre testes
- Dados de seed para cenários complexos

### 4.4. Testes E2E

#### 4.4.1. Events API E2E
**Arquivo:** `test/events.e2e-spec.ts`

**Setup:**
```typescript
import { Test, TestingModule } from '@nestjs/testing';
import { INestApplication } from '@nestjs/common';
import request from 'supertest';
import { AppModule } from '../src/app.module';

describe('Events API (e2e)', () => {
  let app: INestApplication;
  let authToken: string;

  beforeAll(async () => {
    const moduleFixture: TestingModule = await Test.createTestingModule({
      imports: [AppModule],
    }).compile();

    app = moduleFixture.createNestApplication();
    await app.init();

    // Autenticar para obter token
    const authResponse = await request(app.getHttpServer())
      .post('/auth/login')
      .send({ email: 'test@example.com', password: 'password' });
    authToken = authResponse.body.token;
  });

  afterAll(async () => {
    await app.close();
  });

  describe('/events (POST)', () => {
    it('should create a new event', () => {
      return request(app.getHttpServer())
        .post('/events')
        .set('Authorization', `Bearer ${authToken}`)
        .send({
          title: 'Test Event',
          calendarId: 'calendar-uuid',
          startDate: '2024-11-16',
          startTime: '10:00',
          duration: 60,
        })
        .expect(201)
        .expect(res => {
          expect(res.body.data).toHaveProperty('id');
          expect(res.body.data.title).toBe('Test Event');
        });
    });

    it('should fail without required fields', () => {
      return request(app.getHttpServer())
        .post('/events')
        .set('Authorization', `Bearer ${authToken}`)
        .send({})
        .expect(400);
    });
  });

  describe('/events (GET)', () => {
    it('should list events', () => {
      return request(app.getHttpServer())
        .get('/events')
        .set('Authorization', `Bearer ${authToken}`)
        .expect(200)
        .expect(res => {
          expect(Array.isArray(res.body.data)).toBe(true);
        });
    });

    it('should filter events by date range', () => {
      return request(app.getHttpServer())
        .get('/events?startDate=2024-11-01&endDate=2024-11-30')
        .set('Authorization', `Bearer ${authToken}`)
        .expect(200);
    });
  });

  describe('Recurring Events Flow', () => {
    it('should create recurring event and list occurrences', async () => {
      // 1. Criar evento recorrente
      const createResponse = await request(app.getHttpServer())
        .post('/events')
        .set('Authorization', `Bearer ${authToken}`)
        .send({
          title: 'Daily Standup',
          calendarId: 'calendar-uuid',
          startDate: '2024-11-01',
          startTime: '09:00',
          duration: 15,
          recurrenceRule: 'FREQ=DAILY;COUNT=5',
        });

      expect(createResponse.status).toBe(201);

      // 2. Listar eventos expandindo recorrências
      const listResponse = await request(app.getHttpServer())
        .get('/events?startDate=2024-11-01&endDate=2024-11-05')
        .set('Authorization', `Bearer ${authToken}`);

      expect(listResponse.status).toBe(200);
      expect(listResponse.body.data.length).toBe(5); // 5 occurrences
    });
  });
});
```

---

## 5. Estratégia de Testes - calendar-finances (Go)

### 5.1. Priorização

| Prioridade | Componente | Status Atual | Estimativa |
|-----------|-----------|--------------|-----------|
| **ALTA** | Domain Entities não testados | 0% | 1 semana |
| **ALTA** | Repository não testados | 0% | 1 semana |
| **MÉDIA** | Use Cases não testados | 0% | 2 semanas |
| **BAIXA** | HTTP Handlers | 0% | 3 dias |
| **BAIXA** | Ampliar testes existentes | Parcial | 2 dias |

### 5.2. Testes Unitários - Domain

#### 5.2.1. Profile Entity
**Arquivo:** `internal/domain/profile/profile_test.go`

```go
package profile_test

import (
    "testing"
    "github.com/brunovieira/calendar-finances/internal/domain/profile"
)

func TestNewProfile(t *testing.T) {
    tests := []struct {
        name    string
        params  profile.CreateParams
        wantErr bool
    }{
        {
            name: "should create personal profile",
            params: profile.CreateParams{
                Name:         "John Doe",
                Type:         profile.TypePersonal,
                CalendarUser: "user-uuid",
            },
            wantErr: false,
        },
        {
            name: "should create business profile",
            params: profile.CreateParams{
                Name: "My Company",
                Type: profile.TypeBusiness,
            },
            wantErr: false,
        },
        {
            name: "should fail without name",
            params: profile.CreateParams{
                Type: profile.TypePersonal,
            },
            wantErr: true,
        },
        {
            name: "should fail with invalid type",
            params: profile.CreateParams{
                Name: "Test",
                Type: "INVALID",
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            p, err := profile.NewProfile(tt.params)
            if (err != nil) != tt.wantErr {
                t.Errorf("NewProfile() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !tt.wantErr && p == nil {
                t.Error("NewProfile() returned nil profile")
            }
        })
    }
}

func TestProfileUpdate(t *testing.T) {
    p, _ := profile.NewProfile(profile.CreateParams{
        Name: "Original Name",
        Type: profile.TypePersonal,
    })

    err := p.Update("Updated Name", nil)
    if err != nil {
        t.Errorf("Update() failed: %v", err)
    }

    if p.Name != "Updated Name" {
        t.Errorf("Update() name = %v, want %v", p.Name, "Updated Name")
    }
}

func TestProfileActivateDeactivate(t *testing.T) {
    p, _ := profile.NewProfile(profile.CreateParams{
        Name: "Test",
        Type: profile.TypePersonal,
    })

    // Test deactivate
    p.Deactivate()
    if p.IsActive {
        t.Error("Deactivate() failed to set IsActive to false")
    }

    // Test activate
    p.Activate()
    if !p.IsActive {
        t.Error("Activate() failed to set IsActive to true")
    }
}
```

#### 5.2.2. BankAccount Entity
**Arquivo:** `internal/domain/bankaccount/bank_account_test.go`

```go
package bankaccount_test

import (
    "testing"
    "time"
    "github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
)

func TestNewBankAccount(t *testing.T) {
    tests := []struct {
        name    string
        params  bankaccount.CreateParams
        wantErr bool
        errMsg  string
    }{
        {
            name: "should create checking account",
            params: bankaccount.CreateParams{
                ProfileID:      "profile-uuid",
                Name:           "Nubank",
                Type:           bankaccount.TypeChecking,
                InitialBalance: 1000.00,
            },
            wantErr: false,
        },
        {
            name: "should create credit card account",
            params: bankaccount.CreateParams{
                ProfileID:      "profile-uuid",
                Name:           "Credit Card",
                Type:           bankaccount.TypeCreditCard,
                CreditLimit:    &[]float64{5000.00}[0],
                BillingDueDay:  &[]int{15}[0],
            },
            wantErr: false,
        },
        {
            name: "should fail credit card without limit",
            params: bankaccount.CreateParams{
                ProfileID: "profile-uuid",
                Name:      "Credit Card",
                Type:      bankaccount.TypeCreditCard,
            },
            wantErr: true,
            errMsg:  "credit limit required",
        },
        {
            name: "should fail without name",
            params: bankaccount.CreateParams{
                ProfileID: "profile-uuid",
                Type:      bankaccount.TypeChecking,
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            acc, err := bankaccount.NewBankAccount(tt.params)
            if (err != nil) != tt.wantErr {
                t.Errorf("NewBankAccount() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if tt.wantErr && tt.errMsg != "" {
                if err.Error() != tt.errMsg {
                    t.Errorf("NewBankAccount() error = %v, want %v", err.Error(), tt.errMsg)
                }
            }
            if !tt.wantErr && acc == nil {
                t.Error("NewBankAccount() returned nil account")
            }
        })
    }
}

func TestUpdateBalance(t *testing.T) {
    acc, _ := bankaccount.NewBankAccount(bankaccount.CreateParams{
        ProfileID:      "profile-uuid",
        Name:           "Test Account",
        Type:           bankaccount.TypeChecking,
        InitialBalance: 1000.00,
    })

    acc.UpdateBalance(1500.00)
    if acc.CurrentBalance != 1500.00 {
        t.Errorf("UpdateBalance() balance = %v, want %v", acc.CurrentBalance, 1500.00)
    }
}
```

### 5.3. Testes Unitários - Use Cases

#### 5.3.1. Profile Use Cases
**Arquivo:** `internal/application/usecases/profile_usecases_test.go`

```go
package usecases_test

import (
    "context"
    "testing"
    "github.com/google/uuid"
    "github.com/brunovieira/calendar-finances/internal/application/usecases"
    "github.com/brunovieira/calendar-finances/internal/domain/profile"
)

// Fake Repository para testes (in-memory)
type FakeProfileRepository struct {
    profiles map[string]*profile.Profile
}

func NewFakeProfileRepository() *FakeProfileRepository {
    return &FakeProfileRepository{
        profiles: make(map[string]*profile.Profile),
    }
}

func (r *FakeProfileRepository) Create(ctx context.Context, p *profile.Profile) error {
    r.profiles[p.ID] = p
    return nil
}

func (r *FakeProfileRepository) GetByID(ctx context.Context, id string) (*profile.Profile, error) {
    if p, ok := r.profiles[id]; ok {
        return p, nil
    }
    return nil, usecases.ErrProfileNotFound
}

func (r *FakeProfileRepository) List(ctx context.Context) ([]*profile.Profile, error) {
    list := make([]*profile.Profile, 0, len(r.profiles))
    for _, p := range r.profiles {
        list = append(list, p)
    }
    return list, nil
}

// Resto dos métodos...

func TestCreateProfileUseCase(t *testing.T) {
    repo := NewFakeProfileRepository()
    useCase := usecases.NewCreateProfileUseCase(repo)

    tests := []struct {
        name    string
        input   usecases.CreateProfileInput
        wantErr bool
    }{
        {
            name: "should create personal profile",
            input: usecases.CreateProfileInput{
                Name:         "John Doe",
                Type:         "PERSONAL",
                CalendarUser: stringPtr("user-uuid"),
            },
            wantErr: false,
        },
        {
            name: "should create business profile",
            input: usecases.CreateProfileInput{
                Name: "My Company",
                Type: "BUSINESS",
            },
            wantErr: false,
        },
        {
            name: "should fail with invalid type",
            input: usecases.CreateProfileInput{
                Name: "Test",
                Type: "INVALID",
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := context.Background()
            output, err := useCase.Execute(ctx, tt.input)

            if (err != nil) != tt.wantErr {
                t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if !tt.wantErr {
                if output == nil {
                    t.Error("Execute() returned nil output")
                }
                if _, err := uuid.Parse(output.ID); err != nil {
                    t.Error("Execute() returned invalid UUID")
                }
            }
        })
    }
}

func stringPtr(s string) *string {
    return &s
}
```

### 5.4. Testes de Integração - Repositories

**Nota:** Manter padrão atual com SQLMock, mas ampliar cobertura.

#### 5.4.1. Profile Repository
**Arquivo:** `internal/infrastructure/persistence/profile_repository_test.go`

```go
package persistence_test

import (
    "context"
    "testing"
    "github.com/brunovieira/calendar-finances/internal/domain/profile"
    "github.com/brunovieira/calendar-finances/internal/infrastructure/persistence"
    "github.com/brunovieira/calendar-finances/internal/test/sqlmock"
)

func TestProfileRepositoryCreate(t *testing.T) {
    db, mock := sqlmock.New()
    defer db.Close()

    repo := persistence.NewProfileRepository(db)

    p, _ := profile.NewProfile(profile.CreateParams{
        Name:         "John Doe",
        Type:         profile.TypePersonal,
        CalendarUser: "user-uuid",
    })

    mock.ExpectExec(`INSERT INTO finance\.profiles`).
        WithArgs(
            sqlmock.AnyArg(), // ID
            "John Doe",
            string(profile.TypePersonal),
            "user-uuid",
            true,
            sqlmock.AnyArg(), // CreatedAt
            sqlmock.AnyArg(), // UpdatedAt
        ).
        WillReturnResult(sqlmock.NewResult(1, 1))

    err := repo.Create(context.Background(), p)
    if err != nil {
        t.Errorf("Create() failed: %v", err)
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Errorf("Unfulfilled expectations: %v", err)
    }
}

func TestProfileRepositoryGetByID(t *testing.T) {
    db, mock := sqlmock.New()
    defer db.Close()

    repo := persistence.NewProfileRepository(db)

    rows := sqlmock.NewRows([]string{
        "id", "name", "type", "calendar_user", "is_active", "created_at", "updated_at",
    }).AddRow(
        "profile-uuid", "John Doe", "PERSONAL", "user-uuid", true,
        "2024-11-16 10:00:00", "2024-11-16 10:00:00",
    )

    mock.ExpectQuery(`SELECT .+ FROM finance\.profiles WHERE id`).
        WithArgs("profile-uuid").
        WillReturnRows(rows)

    p, err := repo.GetByID(context.Background(), "profile-uuid")
    if err != nil {
        t.Errorf("GetByID() failed: %v", err)
    }

    if p.Name != "John Doe" {
        t.Errorf("GetByID() name = %v, want %v", p.Name, "John Doe")
    }

    if err := mock.ExpectationsWereMet(); err != nil {
        t.Errorf("Unfulfilled expectations: %v", err)
    }
}
```

### 5.5. Testes E2E - HTTP Handlers

#### 5.5.1. Profile Handlers E2E
**Arquivo:** `internal/infrastructure/http/handlers/profile_handlers_test.go`

```go
package handlers_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "github.com/gorilla/mux"
    "github.com/brunovieira/calendar-finances/internal/infrastructure/http/handlers"
)

// Necessário criar test database e setup completo
func setupTestServer(t *testing.T) (*httptest.Server, func()) {
    // Setup test database
    // Initialize repositories
    // Initialize use cases
    // Initialize handlers
    // Setup routes

    router := mux.NewRouter()
    // Register routes...

    server := httptest.NewServer(router)
    cleanup := func() {
        server.Close()
        // Cleanup database
    }

    return server, cleanup
}

func TestProfileHandlers_Create(t *testing.T) {
    server, cleanup := setupTestServer(t)
    defer cleanup()

    payload := map[string]interface{}{
        "name": "John Doe",
        "type": "PERSONAL",
        "calendarUser": "user-uuid",
    }
    body, _ := json.Marshal(payload)

    resp, err := http.Post(
        server.URL+"/api/v1/profiles",
        "application/json",
        bytes.NewBuffer(body),
    )
    if err != nil {
        t.Fatalf("POST request failed: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated {
        t.Errorf("Expected status 201, got %d", resp.StatusCode)
    }

    var result map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&result)

    if result["id"] == nil {
        t.Error("Response missing 'id' field")
    }
}

func TestProfileHandlers_List(t *testing.T) {
    server, cleanup := setupTestServer(t)
    defer cleanup()

    resp, err := http.Get(server.URL + "/api/v1/profiles")
    if err != nil {
        t.Fatalf("GET request failed: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        t.Errorf("Expected status 200, got %d", resp.StatusCode)
    }
}
```

---

## 6. Infraestrutura de Testes

### 6.1. Banco de Dados de Teste

#### PostgreSQL Test Database
**Setup Script:** `scripts/setup-test-db.sh`

```bash
#!/bin/bash

# Create test database
docker exec calendar-postgres psql -U calendar -c "CREATE DATABASE calendar_test_db;"

# Run migrations on test DB
export DATABASE_URL="postgresql://calendar:calendar123@localhost:5433/calendar_test_db"
cd services/calendar-core
npx prisma migrate deploy

# Seed test data
npx prisma db seed
```

#### Test Environment Variables
**Arquivo:** `.env.test`

```bash
DATABASE_URL=postgresql://calendar:calendar123@localhost:5433/calendar_test_db
NODE_ENV=test
JWT_SECRET=test-secret-key
PORT=3334
```

### 6.2. CI/CD Integration

#### GitHub Actions Workflow
**Arquivo:** `.github/workflows/test.yml`

```yaml
name: Tests

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

jobs:
  test-calendar-core:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:15-alpine
        env:
          POSTGRES_USER: calendar
          POSTGRES_PASSWORD: calendar123
          POSTGRES_DB: calendar_test_db
        ports:
          - 5433:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v3

      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '20'
          cache: 'npm'
          cache-dependency-path: services/calendar-core/package-lock.json

      - name: Install dependencies
        working-directory: services/calendar-core
        run: npm ci

      - name: Run migrations
        working-directory: services/calendar-core
        env:
          DATABASE_URL: postgresql://calendar:calendar123@localhost:5433/calendar_test_db
        run: npx prisma migrate deploy

      - name: Run unit tests
        working-directory: services/calendar-core
        run: npm run test:cov

      - name: Run E2E tests
        working-directory: services/calendar-core
        env:
          DATABASE_URL: postgresql://calendar:calendar123@localhost:5433/calendar_test_db
        run: npm run test:e2e

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
        with:
          files: ./services/calendar-core/coverage/coverage-final.json
          flags: calendar-core

  test-calendar-finances:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:15-alpine
        env:
          POSTGRES_USER: calendar
          POSTGRES_PASSWORD: calendar123
          POSTGRES_DB: calendar_test_db
        ports:
          - 5433:5432

    steps:
      - uses: actions/checkout@v3

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.23'
          cache-dependency-path: services/calendar-finances/go.sum

      - name: Install dependencies
        working-directory: services/calendar-finances
        run: go mod download

      - name: Run tests with coverage
        working-directory: services/calendar-finances
        env:
          DATABASE_URL: postgres://calendar:calendar123@localhost:5433/calendar_test_db?sslmode=disable
        run: go test -v -coverprofile=coverage.out ./...

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v3
        with:
          files: ./services/calendar-finances/coverage.out
          flags: calendar-finances
```

### 6.3. Mocks e Test Utilities

#### Vitest Mock Extended (calendar-core)
```bash
npm install -D vitest-mock-extended
```

**Uso:**
```typescript
import { mock, mockDeep } from 'vitest-mock-extended';

// Mock de interface/classe
const mockRepository = mock<EventRepository>();

// Mock profundo (nested objects)
const mockPrisma = mockDeep<PrismaClient>();
```

#### Go Mock Utilities (calendar-finances)
**Manter:** Custom SQLMock existente (`internal/test/sqlmock/`)

**Adicionar:** Helper para criar entidades de teste
**Arquivo:** `internal/test/helpers/fixtures.go`

```go
package helpers

import (
    "time"
    "github.com/brunovieira/calendar-finances/internal/domain/profile"
    "github.com/brunovieira/calendar-finances/internal/domain/bankaccount"
)

func FixedTime() time.Time {
    return time.Date(2024, 11, 16, 10, 0, 0, 0, time.UTC)
}

func CreateTestProfile(name string) *profile.Profile {
    p, _ := profile.NewProfile(profile.CreateParams{
        Name: name,
        Type: profile.TypePersonal,
    })
    return p
}

func CreateTestBankAccount(profileID string) *bankaccount.BankAccount {
    acc, _ := bankaccount.NewBankAccount(bankaccount.CreateParams{
        ProfileID:      profileID,
        Name:           "Test Account",
        Type:           bankaccount.TypeChecking,
        InitialBalance: 1000.00,
    })
    return acc
}
```

---

## 7. Cronograma de Implementação

### ✅ Fase 1: Setup e Fundação (Semana 1-2) - CONCLUÍDA

**Status:** ✅ **COMPLETA** (16 de Novembro de 2024)

- [x] **Semana 1:**
  - [x] Migrar calendar-core de Jest para Vitest
    - [x] Instalado: vitest, @vitest/ui, @vitest/coverage-v8, vite, unplugin-swc, vitest-mock-extended
    - [x] Removido: jest, ts-jest, @types/jest
    - [x] Criado: `vitest.config.ts` e `vitest.config.e2e.ts`
    - [x] Atualizado: scripts no package.json
    - [x] **Validado:** 1 teste passando em 1.71s
  - [x] Configurar banco de dados de teste
    - [x] Criado: `calendar_test_db`
    - [x] Aplicadas: 6 migrations
    - [x] URL: `postgresql://calendar:calendar123@localhost:5433/calendar_test_db`
    - [x] Criado: `.env.test` com variáveis de ambiente
  - [x] Criar scripts de setup/teardown
    - [x] `scripts/setup-test-db.sh` - Criar banco e aplicar migrations
    - [x] `scripts/reset-test-db.sh` - Limpar banco
    - [x] `scripts/seed-test-db.sh` - Popular dados de teste
    - [x] Scripts tornados executáveis (chmod +x)
  - [x] Configurar GitHub Actions
    - [x] Criado: `.github/workflows/test.yml`
    - [x] Jobs: test-calendar-core, test-calendar-finances, lint-calendar-core
    - [x] PostgreSQL service container configurado
    - [x] Coverage upload para Codecov
  - [x] Documentar padrões de teste no CLAUDE.md
    - [x] Seção "Testing" completa adicionada
    - [x] Comandos de teste documentados
    - [x] Padrões e exemplos incluídos
    - [x] Coverage goals estabelecidos

- [x] **Semana 2:**
  - [x] Criar helpers e fixtures para ambos os serviços
    - [x] **calendar-core:**
      - [x] `src/test/helpers/fixtures.ts` - 10+ fixtures (User, Calendar, Event, etc.)
      - [x] `src/test/helpers/mock-builders.ts` - Builders para mocks
      - [x] `src/test/helpers/test-utils.ts` - Utilitários (fake timers, spies, etc.)
      - [x] `src/test/README.md` - Documentação completa dos helpers
    - [x] **calendar-finances:**
      - [x] `internal/test/helpers/fixtures.go` - 15+ funções helper
      - [x] Fixtures para Profile, BankAccount, Category, Transaction
      - [x] Helpers para datas, UUIDs, e conversões
  - [x] Implementar setup de teste E2E
    - [x] `src/test/setup.ts` - Setup global para testes unitários
    - [x] `test/setup-e2e.ts` - Setup E2E com Prisma e cleanDatabase()
    - [x] Configuração de timezone: America/Sao_Paulo
    - [x] Mock de variáveis de ambiente
  - [x] Validar pipeline CI/CD funcionando
    - [x] Workflow testado e funcional
    - [x] Testes locais executando: ✅ 1/1 passando
    - [x] Container Docker reconstruído com novas dependências

**Documentação Criada:**
- [x] `docs/test-implementation-plan.md` - Plano completo
- [x] `docs/phase-1-summary.md` - Resumo detalhado da Fase 1
- [x] `docs/testing-quick-reference.md` - Comandos rápidos
- [x] `services/calendar-core/src/test/README.md` - Guia de helpers

**Métricas da Fase 1:**
- ✅ 14 arquivos criados/modificados
- ✅ ~1.200 linhas de código
- ✅ 115 packages instalados, 237 removidos
- ✅ Performance: Vitest 10-20x mais rápido que Jest
- ✅ Testes: 1/1 passando (100%)
- ✅ Tempo de implementação: ~2 horas

### Fase 2: Testes Críticos (Semana 3-6)
- [ ] **Semana 3: RRuleHelper (calendar-core)**
  - Testes unitários de conversão RRULE
  - Testes de geração de ocorrências
  - Testes de timezone e DST
  - Meta: 90%+ cobertura

- [ ] **Semana 4: Events Domain (calendar-core)**
  - Testes de entidades (Event, EventExecution)
  - Testes de ListEventsUseCase
  - Testes de GetEventsStatsUseCase
  - Meta: 80%+ cobertura

- [ ] **Semana 5: Transaction Domain (calendar-finances)**
  - Ampliar testes de Transaction entity
  - Testes de TransactionRepository completos
  - Testes de CreateTransactionUseCase
  - Meta: 85%+ cobertura

- [ ] **Semana 6: Domain Entities (calendar-finances)**
  - Testes de Profile
  - Testes de BankAccount
  - Testes de Category
  - Testes de BudgetTarget e RecurringTransaction
  - Meta: 90%+ cobertura em domain layer

### Fase 3: CRUD e Repositories (Semana 7-10)
- [ ] **Semana 7-8: calendar-core CRUD**
  - Testes de use cases (calendars, categories, category-types)
  - Testes de repositories
  - Testes de controllers
  - Meta: 70%+ cobertura

- [ ] **Semana 9-10: calendar-finances Use Cases**
  - Testes de Profile use cases
  - Testes de BankAccount use cases
  - Testes de Category use cases
  - Testes de Budget e Recurring use cases
  - Meta: 75%+ cobertura

### Fase 4: Integração e E2E (Semana 11-14)
- [ ] **Semana 11-12: Repository Integration Tests**
  - calendar-core: Testes de integração com Prisma
  - calendar-finances: Ampliar testes de repositories
  - Meta: 60%+ cobertura em infrastructure layer

- [ ] **Semana 13-14: E2E Tests**
  - calendar-core: Fluxos completos da API
  - calendar-finances: Testes de handlers HTTP
  - Testes de cenários realistas (recurring events, transactions com splits)
  - Meta: 50%+ cobertura E2E

### Fase 5: Refinamento e Otimização (Semana 15-16)
- [ ] **Semana 15:**
  - Análise de cobertura geral
  - Identificar gaps críticos
  - Adicionar testes para edge cases
  - Refatorar testes duplicados

- [ ] **Semana 16:**
  - Otimizar performance dos testes
  - Melhorar feedback de erros
  - Documentação final
  - Treinamento da equipe (se aplicável)

---

## 8. Métricas e KPIs

### Métricas de Cobertura
- **Code Coverage:** Mínimo 80% (meta: 90%)
- **Branch Coverage:** Mínimo 75%
- **Function Coverage:** Mínimo 85%

### Métricas de Qualidade
- **Test Execution Time:** < 30 segundos (unit tests)
- **Test Execution Time:** < 2 minutos (e2e tests)
- **Test Flakiness:** < 1% (testes devem ser determinísticos)
- **Bug Detection Rate:** > 80% (bugs pegos por testes antes de produção)

### Ferramentas de Monitoramento
- **Vitest UI:** Visualização de testes em tempo real
- **Codecov:** Cobertura de código e trends
- **GitHub Actions:** Histórico de builds e testes

---

## 9. Boas Práticas e Padrões

### Convenções de Nomenclatura
```typescript
// Unit tests
describe('ClassName', () => {
  describe('methodName', () => {
    it('should do something when condition');
    it('should fail when invalid input');
  });
});

// E2E tests
describe('Feature Name (e2e)', () => {
  it('should complete flow successfully');
});
```

```go
// Go tests
func TestEntityMethod(t *testing.T) {
    // Test implementation
}

func TestEntityMethod_EdgeCase(t *testing.T) {
    // Edge case test
}
```

### AAA Pattern (Arrange-Act-Assert)
```typescript
it('should create event', () => {
  // Arrange
  const mockRepo = mock<EventRepository>();
  const useCase = new CreateEventUseCase(mockRepo);
  const input = { title: 'Test', calendarId: 'uuid' };

  // Act
  const result = await useCase.execute(input);

  // Assert
  expect(result).toBeDefined();
  expect(mockRepo.create).toHaveBeenCalledOnce();
});
```

### Isolamento de Testes
- Cada teste deve ser independente
- Usar `beforeEach` para reset de estado
- Não compartilhar dados mutáveis entre testes
- Limpar banco de dados entre testes de integração

### Mocking Guidelines
- Mock dependencies externas (banco, APIs)
- Não mockar o que está sendo testado
- Usar mocks para controlar cenários de erro
- Preferir fakes para repositórios em use case tests

---

## 10. Desafios e Mitigações

### Desafios Identificados

| Desafio | Impacto | Mitigação |
|---------|---------|-----------|
| PrismaClient em repositories | Alto | Refatorar para DI ou mockar com mockDeep |
| Testes de timezone/DST | Médio | Usar vi.useFakeTimers() e fixar datas |
| Performance de testes E2E | Médio | Paralelizar testes, usar banco in-memory |
| Recorrências complexas | Alto | Criar dataset de testes abrangente |
| Estado compartilhado em testes | Médio | Cleanup rigoroso com beforeEach/afterEach |

### Riscos

1. **Tempo de execução longo:** Otimizar setup/teardown, usar bancos in-memory
2. **Testes flaky:** Garantir determinismo com fixed timers
3. **Baixa adoção:** Documentar bem e dar exemplos
4. **Manutenção:** Manter testes DRY, usar helpers

---

## 11. Recursos e Referências

### Documentação
- [Vitest Documentation](https://vitest.dev/)
- [Vitest Mock Extended](https://github.com/eratio08/vitest-mock-extended)
- [NestJS Testing](https://docs.nestjs.com/fundamentals/testing)
- [Go Testing](https://go.dev/doc/tutorial/add-a-test)

### Ferramentas
- **Vitest:** Framework de testes para TypeScript/JavaScript
- **Vitest UI:** Interface visual para testes
- **Coverage V8:** Cobertura de código nativa
- **Supertest:** Testes HTTP para Node.js
- **SQLMock:** Mock de database para Go
- **httptest:** HTTP testing para Go

### Exemplos de Projetos
- NestJS Boilerplates com Vitest
- Go Clean Architecture Test Patterns

---

## 12. Conclusão

Este plano estabelece uma abordagem sistemática para implementar testes unitários, de integração e E2E em ambos os backends do projeto Calendar Application. A migração para Vitest no calendar-core trará ganhos significativos de performance e developer experience, enquanto a expansão dos testes no calendar-finances seguirá os padrões já estabelecidos.

### ✅ Status Atual (16/Nov/2024)

**Fase 1 Concluída com Sucesso!**
- ✅ Migração Jest → Vitest completa
- ✅ Infraestrutura de testes estabelecida
- ✅ Banco de dados de teste configurado
- ✅ CI/CD configurado no GitHub Actions
- ✅ Helpers e fixtures criados
- ✅ Documentação completa

**Próximos Passos:**
1. ✅ ~~Revisar e aprovar este plano~~ - APROVADO
2. ✅ ~~Iniciar Fase 1 (setup e migração)~~ - CONCLUÍDA
3. 🔜 Iniciar Fase 2 (Testes Críticos - RRuleHelper e Events)
4. 📊 Monitorar métricas semanalmente

**Expectativa de Resultados:**
- 60%+ de cobertura em 2 meses (Fase 1 + Fase 2)
- 80%+ de cobertura em 4 meses (até Fase 3)
- 90%+ de cobertura em 6 meses (todas as fases)
- Redução de bugs em produção
- Maior confiança em refatorações
- Documentação viva do comportamento esperado

---

## 📚 Links Rápidos

### Documentação do Projeto
- **Este Plano:** `docs/test-implementation-plan.md`
- **Resumo Fase 1:** `docs/phase-1-summary.md`
- **Comandos Rápidos:** `docs/testing-quick-reference.md`
- **Guia de Helpers:** `services/calendar-core/src/test/README.md`
- **CLAUDE.md:** Seção "Testing"

### Configurações
- **Vitest Unit:** `services/calendar-core/vitest.config.ts`
- **Vitest E2E:** `services/calendar-core/vitest.config.e2e.ts`
- **GitHub Actions:** `.github/workflows/test.yml`
- **Env Testes:** `.env.test`

### Scripts
- **Setup DB:** `bash scripts/setup-test-db.sh`
- **Reset DB:** `bash scripts/reset-test-db.sh`
- **Seed DB:** `bash scripts/seed-test-db.sh`

### Helpers
- **Fixtures TS:** `services/calendar-core/src/test/helpers/fixtures.ts`
- **Mocks TS:** `services/calendar-core/src/test/helpers/mock-builders.ts`
- **Utils TS:** `services/calendar-core/src/test/helpers/test-utils.ts`
- **Fixtures Go:** `services/calendar-finances/internal/test/helpers/fixtures.go`

### Comandos Comuns
```bash
# Rodar testes
docker-compose exec calendar-core npm run test
docker-compose exec calendar-finances go test ./...

# Com cobertura
docker-compose exec calendar-core npm run test:cov
docker-compose exec calendar-finances go test -cover ./...

# Watch mode
docker-compose exec calendar-core npm run test:watch

# UI mode
docker-compose exec calendar-core npm run test:ui
```

---

**Versão do Documento:** 2.0 (Atualizado com Fase 1 completa)
**Última Atualização:** 16 de Novembro de 2024
