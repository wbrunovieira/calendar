# Alternativas ao Redis para Projeto Pessoal

## Decisão Arquitetural

Este projeto é pessoal e de usuário único, portanto Redis foi removido para reduzir custos de deploy e complexidade operacional.

## Alternativas Implementadas

### 1. Cache em Memória

**Solução**: NestJS Cache Manager com store em memória

```bash
npm install @nestjs/cache-manager cache-manager
```

**Exemplo de uso**:

```typescript
// app.module.ts
import { CacheModule } from '@nestjs/cache-manager';

@Module({
  imports: [
    CacheModule.register({
      isGlobal: true,
      ttl: 600, // 10 minutos em segundos
      max: 100, // máximo de itens no cache
    }),
  ],
})
export class AppModule {}

// calendar.service.ts
import { CACHE_MANAGER } from '@nestjs/cache-manager';
import { Cache } from 'cache-manager';

@Injectable()
export class CalendarService {
  constructor(@Inject(CACHE_MANAGER) private cacheManager: Cache) {}

  async getEvents(userId: string) {
    const cacheKey = `events:${userId}`;
    const cached = await this.cacheManager.get(cacheKey);

    if (cached) {
      return cached;
    }

    const events = await this.fetchFromDatabase(userId);
    await this.cacheManager.set(cacheKey, events, 600); // TTL 10 min
    return events;
  }
}
```

**Vantagens**:
- ✅ Zero custos adicionais
- ✅ Configuração simples
- ✅ Suficiente para usuário único
- ✅ Sem dependência externa

**Desvantagens**:
- ❌ Cache se perde ao reiniciar (aceitável para projeto pessoal)

---

### 2. Filas de Jobs

**Solução**: BullMQ com PostgreSQL adapter ou pg-boss

#### Opção A: BullMQ com PostgreSQL (Recomendado)

```bash
npm install bullmq ioredis-mock
```

```typescript
// Use PostgreSQL como backend via adapter customizado
// Ou use pg-boss (opção B)
```

#### Opção B: pg-boss (Mais Simples)

```bash
npm install pg-boss
npm install --save-dev @types/pg-boss
```

**Exemplo de uso**:

```typescript
// jobs.module.ts
import PgBoss from 'pg-boss';

@Module({
  providers: [
    {
      provide: 'PG_BOSS',
      useFactory: async () => {
        const boss = new PgBoss({
          connectionString: process.env.DATABASE_URL,
        });
        await boss.start();
        return boss;
      },
    },
    JobsService,
  ],
  exports: ['PG_BOSS'],
})
export class JobsModule {}

// jobs.service.ts
@Injectable()
export class JobsService {
  constructor(@Inject('PG_BOSS') private boss: PgBoss) {}

  async scheduleCalendarSync(userId: string) {
    await this.boss.send('sync-calendar', { userId }, {
      retryLimit: 3,
      retryDelay: 60,
      expireInHours: 1,
    });
  }

  async processJobs() {
    await this.boss.work('sync-calendar', async (job) => {
      const { userId } = job.data;
      // Processar sincronização
      return { success: true };
    });
  }
}
```

**Vantagens**:
- ✅ Usa PostgreSQL existente
- ✅ Zero custos adicionais
- ✅ Persistente (jobs não se perdem)
- ✅ Retry e agendamento built-in
- ✅ Interface simples

**Desvantagens**:
- ❌ Menor performance que Redis (aceitável para uso pessoal)

---

### 3. Sessões e Autenticação

**Solução**: JWT Stateless (sem storage)

```bash
npm install @nestjs/jwt @nestjs/passport passport passport-jwt
npm install --save-dev @types/passport-jwt
```

**Exemplo de uso**:

```typescript
// auth.module.ts
import { JwtModule } from '@nestjs/jwt';

@Module({
  imports: [
    JwtModule.register({
      secret: process.env.JWT_SECRET,
      signOptions: { expiresIn: '7d' },
    }),
  ],
})
export class AuthModule {}

// auth.service.ts
@Injectable()
export class AuthService {
  constructor(private jwtService: JwtService) {}

  async login(user: any) {
    const payload = { email: user.email, sub: user.id };
    return {
      access_token: this.jwtService.sign(payload),
    };
  }
}
```

**Alternativa (se precisar invalidar tokens)**:

Use PostgreSQL para blacklist de tokens revogados:

```sql
CREATE TABLE token_blacklist (
  token VARCHAR(500) PRIMARY KEY,
  revoked_at TIMESTAMP DEFAULT NOW(),
  expires_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_expires_at ON token_blacklist(expires_at);
```

**Vantagens**:
- ✅ Stateless, sem storage necessário
- ✅ Escalável
- ✅ Standard da indústria

**Desvantagens**:
- ❌ Não pode invalidar tokens antes do expiry (use blacklist se necessário)

---

## Comparação de Custos

### Com Redis (Deploy)
- Heroku Redis: $15-50/mês
- Railway Redis: $5-20/mês
- Upstash Redis: $0.20-10/mês (pay-as-go)
- **Total**: $5-50/mês adicional

### Sem Redis (Alternativas)
- Cache em memória: $0
- pg-boss: $0 (usa PostgreSQL existente)
- JWT: $0
- **Total**: $0 adicional

**Economia anual**: $60-600/ano

---

## Quando Considerar Adicionar Redis

Adicione Redis apenas se:
- [ ] Houver mais de 100 usuários ativos simultâneos
- [ ] Necessitar de cache distribuído entre múltiplas instâncias
- [ ] Jobs críticos exigirem processamento com latência < 10ms
- [ ] Rate limiting muito agressivo (milhares de requests/segundo)

Para um projeto pessoal de usuário único, **nenhum desses casos se aplica**.

---

## Próximos Passos

1. ✅ Implementar cache em memória com NestJS Cache Manager
2. ✅ Configurar pg-boss para jobs assíncronos
3. ✅ Implementar autenticação JWT
4. ❌ Monitorar performance e revisar se necessário (improvável)
