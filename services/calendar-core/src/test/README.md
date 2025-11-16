# Test Helpers and Utilities

Este diretório contém helpers, fixtures e utilitários para testes do calendar-core.

## Estrutura

```
src/test/
├── setup.ts              # Setup global de testes
├── README.md            # Este arquivo
└── helpers/
    ├── fixtures.ts      # Dados de teste reutilizáveis
    ├── mock-builders.ts # Builders para mocks
    └── test-utils.ts    # Utilitários gerais
```

## Como Usar

### Fixtures

Use fixtures para criar dados de teste consistentes:

```typescript
import {
  createEventFixture,
  createCalendarFixture,
  createUserFixture,
  RRULE_DAILY,
  fixedTime
} from '@/test/helpers/fixtures';

// Criar evento com valores padrão
const event = createEventFixture();

// Sobrescrever valores específicos
const customEvent = createEventFixture({
  title: 'My Custom Event',
  duration: 120
});

// Usar constantes de RRULE
const recurringEvent = createEventFixture({
  recurrenceRule: RRULE_DAILY
});

// Usar data fixa para testes determinísticos
const testDate = fixedTime(); // 2024-11-16T10:00:00-03:00
```

### Mock Builders

Crie mocks facilmente com os builders:

```typescript
import { createMockRepository, createMockPrisma } from '@/test/helpers/mock-builders';

// Mock de repositório
const mockEventRepo = createMockRepository<EventRepository>();
mockEventRepo.create.mockResolvedValue(eventEntity);

// Mock profundo do Prisma
const mockPrisma = createMockPrisma();
mockPrisma.event.create.mockResolvedValue(prismaEvent);
```

### Test Utilities

Utilitários para controle de testes:

```typescript
import { useFakeTimers, spyOnConsole, expectToThrow } from '@/test/helpers/test-utils';

// Controlar tempo nos testes
useFakeTimers(new Date('2024-11-16'));

// Spy em console (evitar poluir output)
const consoleSpy = spyOnConsole();
// ... código que usa console.log
consoleSpy.restore();

// Testar se função lança erro
await expectToThrow(() => {
  throw new Error('Test error');
}, 'Test error');
```

## Fixtures Disponíveis

### Usuários
- `createUserFixture(overrides?)`

### Calendários
- `createCalendarFixture(overrides?)`

### Categorias
- `createCategoryFixture(overrides?)`
- `createCategoryTypeFixture(overrides?)`

### Eventos
- `createEventFixture(overrides?)` - Evento simples
- `createRecurringEventFixture(overrides?)` - Evento recorrente
- `createEventCompletionFixture(overrides?)` - Completion
- `createRecurrenceExceptionFixture(overrides?)` - Exception

### Constantes
- `RRULE_DAILY` - Recorrência diária
- `RRULE_WEEKLY` - Recorrência semanal (MO, WE, FR)
- `RRULE_MONTHLY` - Recorrência mensal (dia 15)
- `RRULE_YEARLY` - Recorrência anual

### Date Ranges
- `dateRange.november2024` - Novembro completo
- `dateRange.week` - Uma semana
- `dateRange.singleDay` - Um dia

## Exemplo Completo

```typescript
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { mock } from 'vitest-mock-extended';
import {
  createEventFixture,
  createCalendarFixture,
  RRULE_DAILY,
  fixedTime
} from '@/test/helpers/fixtures';
import { createMockRepository } from '@/test/helpers/mock-builders';
import { useFakeTimers } from '@/test/helpers/test-utils';

describe('CreateEventUseCase', () => {
  let useCase: CreateEventUseCase;
  let mockEventRepo: MockProxy<EventRepository>;

  beforeEach(() => {
    // Setup fake timers
    useFakeTimers(fixedTime());

    // Create mocks
    mockEventRepo = createMockRepository<EventRepository>();

    // Create use case
    useCase = new CreateEventUseCase(mockEventRepo);
  });

  it('should create recurring event', async () => {
    // Arrange
    const calendar = createCalendarFixture();
    const eventData = {
      title: 'Daily Standup',
      calendarId: calendar.id,
      startDate: '2024-11-16',
      startTime: '09:00',
      duration: 15,
      recurrenceRule: RRULE_DAILY,
    };

    const expectedEvent = createEventFixture({
      ...eventData,
      id: expect.any(String),
    });

    mockEventRepo.create.mockResolvedValue(expectedEvent);

    // Act
    const result = await useCase.execute(eventData);

    // Assert
    expect(result).toBeDefined();
    expect(result.title).toBe('Daily Standup');
    expect(result.recurrenceRule).toBe(RRULE_DAILY);
    expect(mockEventRepo.create).toHaveBeenCalledOnce();
  });
});
```

## Boas Práticas

1. **Use fixtures ao invés de criar objetos manualmente**
   ```typescript
   // ❌ Evite
   const event = {
     id: '123',
     title: 'Test',
     // ... muitos campos
   };

   // ✅ Prefira
   const event = createEventFixture({ title: 'Test' });
   ```

2. **Fixe datas para testes determinísticos**
   ```typescript
   // ❌ Evite
   const now = new Date(); // Muda a cada execução

   // ✅ Prefira
   useFakeTimers(fixedTime()); // Sempre a mesma data
   ```

3. **Limpe mocks após cada teste**
   ```typescript
   afterEach(() => {
     vi.clearAllMocks();
   });
   ```

4. **Use TypeScript para type safety**
   ```typescript
   // Mock tipado
   const mockRepo = createMockRepository<EventRepository>();
   mockRepo.create // autocomplete funciona!
   ```

## Troubleshooting

### "Cannot find module '@/test/helpers/fixtures'"

Verifique se o alias está configurado em `vitest.config.ts`:
```typescript
resolve: {
  alias: {
    '@': resolve(__dirname, './src'),
  },
}
```

### "mockResolvedValue is not a function"

Certifique-se de importar de `vitest-mock-extended`:
```typescript
import { mock } from 'vitest-mock-extended';
```

### Testes com datas falhando intermitentemente

Use `useFakeTimers()` para fixar a data:
```typescript
import { useFakeTimers, fixedTime } from '@/test/helpers/test-utils';

beforeEach(() => {
  useFakeTimers(fixedTime());
});
```

## Adicionar Novos Helpers

Ao adicionar novos helpers, siga o padrão:

1. **Fixtures** (`fixtures.ts`): Dados de teste
2. **Mock Builders** (`mock-builders.ts`): Criação de mocks
3. **Test Utils** (`test-utils.ts`): Utilitários gerais

Documente o helper neste README!
