/**
 * @vitest-environment happy-dom
 */
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, waitFor, cleanup } from '@testing-library/react';
import { ProfileProvider, useProfile } from '@/contexts/ProfileContext';
import { resetSessionState, reportRequestFailure } from '@/lib/session';
import { AuthRequiredError, ApiError, apiBase } from '@/lib/api';

/**
 * The three files that constitute the user-visible fix — the provider, the
 * classifier wiring and the gate that swaps page content — had no coverage at
 * all while the suite ran node-only. Deleting any of them left every test
 * green. These exercise them through the real provider.
 */

function Probe() {
  const { authRequired, loginUrl, loadError, profiles, isLoading } = useProfile();
  if (isLoading) return <div>carregando</div>;
  if (authRequired) return <div data-testid="gate">sessao expirada: {loginUrl}</div>;
  if (loadError) return <div data-testid="erro">falha de carga</div>;
  return <div data-testid="dados">perfis: {profiles.length}</div>;
}

const LOGIN = 'https://auth.wbdigitalsolutions.com/?rd=x';

function mockFetch(impl: () => Promise<unknown>) {
  vi.stubGlobal(
    'fetch',
    vi.fn(async () => {
      const body = await impl();
      return {
        ok: true,
        status: 200,
        statusText: 'OK',
        // Derived, never hardcoded: a fake that pins the origin turns into a
        // cross-origin response the moment NEXT_PUBLIC_API_URL differs, and the
        // test then fails for a reason that has nothing to do with the fix.
        url: `${apiBase()}/profiles`,
        redirected: false,
        headers: { get: () => 'application/json' },
        json: async () => body,
        text: async () => JSON.stringify(body),
      } as unknown as Response;
    }),
  );
}

// Node exposes its own `localStorage` global that is undefined unless the
// runtime was started with --localstorage-file, and it shadows the jsdom one.
// An in-memory stub keeps the test independent of which of the two wins.
function stubLocalStorage() {
  const store = new Map<string, string>();
  vi.stubGlobal('localStorage', {
    getItem: (k: string) => store.get(k) ?? null,
    setItem: (k: string, v: string) => void store.set(k, v),
    removeItem: (k: string) => void store.delete(k),
    clear: () => store.clear(),
  });
}

beforeEach(() => {
  resetSessionState();
  stubLocalStorage();
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  resetSessionState();
});

describe('the session gate', () => {
  it('shows the data when the session is good', async () => {
    mockFetch(async () => ({ data: [{ id: 'a', name: 'Bruno Pessoal' }] }));

    render(
      <ProfileProvider>
        <Probe />
      </ProfileProvider>,
    );

    await waitFor(() => expect(screen.getByTestId('dados').textContent).toContain('perfis: 1'));
  });

  // The regression guard, now at the level the user actually sees: an account
  // that really has no profiles must not be told to log in.
  it('shows the empty state, not the gate, when there genuinely are no profiles', async () => {
    mockFetch(async () => ({ data: [] }));

    render(
      <ProfileProvider>
        <Probe />
      </ProfileProvider>,
    );

    await waitFor(() => expect(screen.getByTestId('dados').textContent).toContain('perfis: 0'));
    expect(screen.queryByTestId('gate')).toBeNull();
  });

  it('raises the gate when the load is an authentication challenge', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => {
        throw new AuthRequiredError(401, 'Unauthorized', LOGIN);
      }),
    );

    render(
      <ProfileProvider>
        <Probe />
      </ProfileProvider>,
    );

    await waitFor(() => expect(screen.getByTestId('gate').textContent).toContain(LOGIN));
  });

  // S4: a server fault is not an empty account either.
  it('reports a load failure separately from an empty account', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => {
        throw new ApiError(500, 'Server Error');
      }),
    );

    render(
      <ProfileProvider>
        <Probe />
      </ProfileProvider>,
    );

    await waitFor(() => expect(screen.getByTestId('erro')).toBeTruthy());
    expect(screen.queryByTestId('gate')).toBeNull();
  });

  // S2: the session that lapses while the tab is open. The mount effect has
  // already run; only a shared, subscribable state can notice this.
  it('raises the gate when a later request discovers the session ended', async () => {
    mockFetch(async () => ({ data: [{ id: 'a', name: 'Bruno Pessoal' }] }));

    render(
      <ProfileProvider>
        <Probe />
      </ProfileProvider>,
    );
    await waitFor(() => expect(screen.getByTestId('dados')).toBeTruthy());

    // Any other call in the app — saving a transaction, paying an invoice.
    reportRequestFailure(new AuthRequiredError(401, 'Unauthorized', LOGIN));

    await waitFor(() => expect(screen.getByTestId('gate').textContent).toContain(LOGIN));
  });

  it('does not raise the gate when a later request merely fails', async () => {
    mockFetch(async () => ({ data: [{ id: 'a', name: 'Bruno Pessoal' }] }));

    render(
      <ProfileProvider>
        <Probe />
      </ProfileProvider>,
    );
    await waitFor(() => expect(screen.getByTestId('dados')).toBeTruthy());

    reportRequestFailure(new ApiError(500, 'Server Error'));

    await waitFor(() => expect(screen.getByTestId('dados')).toBeTruthy());
    expect(screen.queryByTestId('gate')).toBeNull();
  });
});
