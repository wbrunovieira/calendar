# Calendar Frontend

Next.js 15.5 frontend application for the Calendar project.

## Tech Stack

- **Framework**: Next.js 15.5.4 (App Router)
- **React**: 19.1.0
- **TypeScript**: 5.x
- **Styling**: Tailwind CSS 4.x
- **Linting**: ESLint 9.x

## Getting Started

### Prerequisites

- Node.js 18+ installed locally
- Backend service running on `http://localhost:3334`

### Development

```bash
# Install dependencies
npm install

# Run development server
npm run dev
```

Open [http://localhost:3000](http://localhost:3000) in your browser.

### Available Scripts

```bash
# Start development server with hot-reload
npm run dev

# Build for production
npm run build

# Start production server
npm start

# Run linter
npm run lint
```

## Project Structure

```
calendar-frontend/
├── src/
│   ├── app/              # Next.js App Router pages
│   │   ├── page.tsx      # Home page
│   │   └── layout.tsx    # Root layout
│   └── lib/              # Utilities and API client
│       └── api.ts        # Backend API client
├── public/               # Static assets
├── .env.local           # Local environment variables (not committed)
├── .env.example         # Environment variables template
└── package.json
```

## Environment Variables

Copy `.env.example` to `.env.local` and configure:

```bash
CALENDAR_API_URL=http://localhost:3334
CALENDAR_API_TOKEN=            # must match calendar-core API_TOKEN, if set
```

The browser never talks to calendar-core directly: it calls `/api/backend/*` on this app, and the
route handler in `src/app/api/backend/[...path]` forwards the request and adds the token on the
server. Both variables are server-only, so the token is not part of the JS bundle.

## Login

`src/middleware.ts` requires a session on every page and API route except `/login`. Logging in
takes the password and a 6-digit code from an authenticator app (TOTP); the session is a signed,
host-only `__Host-session` cookie (30 days by default, `AUTH_SESSION_DAYS`). Failed logins are
throttled per IP and overall, and a TOTP code works only once.

Set it up with `npm run auth:setup`, which prints `AUTH_PASSWORD_HASH`, `AUTH_TOTP_SECRET`,
`AUTH_SESSION_SECRET` and an `otpauth://` URI for the authenticator app. With no
`AUTH_SESSION_SECRET` the login is off. Changing `AUTH_SESSION_SECRET` signs every session out.

## API Integration

The frontend connects to the backend via the API client in `src/lib/api.ts`.

Example usage:

```typescript
import { api } from '@/lib/api';

// Health check
const health = await api.health();
```

## Design Decisions

### Why Local Development (Not Docker)?

This frontend runs **locally** during development for:
- ⚡ Instant hot-reload
- 🚀 Better performance
- 🛠️ Easier debugging with browser DevTools
- 📦 Faster npm installs

For production, use Docker (Dockerfile to be created when needed).

## Deployment

Production deployment will use Docker. Dockerfile will be created in Phase 3 of development.

## Notes

- This is part of a personal calendar application
- Backend runs in Docker on port 3334
- Frontend development runs locally on port 3000
- TypeScript strict mode enabled
- ESLint configured for Next.js best practices
