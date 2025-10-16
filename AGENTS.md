# Repository Guidelines

## Project Structure & Module Organization
The project is service-oriented: `services/calendar-core/` hosts the NestJS API with Prisma schema under `prisma/` and feature modules in `src/`. `services/calendar-frontend/` and `services/finances-frontend/` are Next.js apps organised by `src/app`, `components/`, and shared utilities. `services/calendar-finances/` is a Go 1.23 service split into `cmd/api/` for entrypoints and `internal/` for handlers, database access, and domain logic. Reference `docs/` for architectural notes, database diagrams, and planning context, and use `docker-compose.yml` when orchestrating multiple services locally.

## Build, Test, and Development Commands
API (`services/calendar-core`):
```bash
npm ci
npm run start:dev        # hot-reload API
npm run test             # unit tests
npm run test:cov         # coverage report
```
Calendar web (`services/calendar-frontend`):
```bash
npm ci
npm run dev              # Next.js dev server on 3000
npm run build && npm run start
npm run lint             # ESLint flat config
```
Finances web (`services/finances-frontend`):
```bash
npm ci
npm run dev              # runs on 3003 by default
npm run lint
```
Finances API (`services/calendar-finances`):
```bash
go mod download
go run cmd/api/main.go
go test ./...
```
To launch shared dependencies (Postgres, Redis, services) run `docker-compose up -d` from the repo root.

## Coding Style & Naming Conventions
TypeScript services follow ESLint + Prettier defaults (2-space indent, single quotes, trailing commas where safe). Keep NestJS files aligned with framework naming (`*.module.ts`, `*.service.ts`, `*.controller.ts`) and use PascalCase for classes, camelCase for variables, and kebab-case for filenames. Next.js components live in `src/components` and should be PascalCase with co-located hooks in `src/hooks`. Run `npm run lint` or `npm run format` (core) before committing. Go code must remain `gofmt`-clean with package-level names in PascalCase and locals in camelCase.

## Testing Guidelines
NestJS uses Jest with `*.spec.ts` files placed in `test/` or alongside sources; aim to extend coverage when touching modules and verify via `npm run test:cov`. Add integration tests around Prisma services when modifying database interactions. Go tests should live next to the code as `*_test.go` and exercise handler/database boundaries with table-driven cases. The frontends currently rely on linting; when adding UI tests use the `__tests__/` pattern and document the command alongside the PR.

## Commit & Pull Request Guidelines
Follow the existing conventional prefixes (`feat:`, `fix:`, `chore:`, etc.) observed in `git log --oneline`. Keep subject lines imperative and under ~72 characters, and group related changes per commit. Pull requests should include a concise summary, linked issue or context, test/lint results (`npm run test`, `go test`, `npm run lint`), and UI screenshots or recordings when altering frontend behaviour. Update relevant docs in `docs/` or README files whenever workflows or schemas change.

## Environment & Configuration Tips
Copy service-specific `.env.example` files where available and avoid committing secrets. Prisma migrations and seeds live under `services/calendar-core/prisma`; after schema updates run `npx prisma migrate dev` and `npm run prisma:seed`. Go service ports default to 3335, while Next apps use 3000 and 3003—adjust `.env` or compose overrides if ports clash. Use `docker-compose logs -f <service>` to inspect runtime issues during local orchestration.
