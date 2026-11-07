/**
 * @vitest-environment happy-dom
 */
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, cleanup } from '@testing-library/react';

/**
 * The gate itself. The provider-level tests prove `authRequired` is computed
 * correctly, but deleting the one line in AppLayout that acts on it left the
 * whole suite green — the flag was asserted, the behaviour it drives was not.
 */

vi.mock('next/navigation', () => ({ usePathname: () => '/' }));

const profileState = {
  profiles: [] as Array<{ id: string; name: string; type: string }>,
  selectedProfileId: null as string | null,
  selectedProfile: null as { id: string; name: string; type: string } | null,
  setSelectedProfileId: () => {},
  isLoading: false,
  authRequired: false,
  loginUrl: '',
  loadError: false,
};

vi.mock('@/contexts/ProfileContext', () => ({
  useProfile: () => profileState,
  ProfileProvider: ({ children }: { children: React.ReactNode }) => children,
}));

import AppLayout from './AppLayout';

beforeEach(() => {
  Object.assign(profileState, { authRequired: false, loginUrl: '', profiles: [] });
});

afterEach(cleanup);

describe('AppLayout', () => {
  it('renders the page when the session is good', () => {
    render(
      <AppLayout>
        <div data-testid="conteudo">lançamentos</div>
      </AppLayout>,
    );

    expect(screen.getByTestId('conteudo')).toBeTruthy();
    expect(screen.queryByText('Sessão expirada')).toBeNull();
  });

  it('replaces the page with the login panel when the session is gone', () => {
    Object.assign(profileState, {
      authRequired: true,
      loginUrl: 'https://auth.wbdigitalsolutions.com/?rd=x',
    });

    render(
      <AppLayout>
        <div data-testid="conteudo">lançamentos</div>
      </AppLayout>,
    );

    // Replaced, not merely annotated: the data behind it was never received,
    // and an empty dashboard under a warning is how this hid in the first place.
    expect(screen.queryByTestId('conteudo')).toBeNull();
    expect(screen.getByText('Sessão expirada')).toBeTruthy();
  });

  it('points the panel and the header at the login URL', () => {
    Object.assign(profileState, {
      authRequired: true,
      loginUrl: 'https://auth.wbdigitalsolutions.com/?rd=x',
    });

    render(
      <AppLayout>
        <div />
      </AppLayout>,
    );

    const links = Array.from(document.querySelectorAll('a'))
      .map((a) => a.getAttribute('href'))
      .filter((h): h is string => h !== null);
    expect(links.filter((h) => h === 'https://auth.wbdigitalsolutions.com/?rd=x').length).toBe(2);

    // Asserting only that "Entrar" appears is not enough: the panel's own
    // "Entrar novamente" satisfies it, so the header could still read "Sair"
    // while the user is logged out. The absence of "Sair" is the real claim.
    expect(screen.getByText('Entrar')).toBeTruthy();
    expect(screen.queryByText('Sair')).toBeNull();
  });

  it('offers a way out when the session is good', () => {
    render(
      <AppLayout>
        <div />
      </AppLayout>,
    );

    expect(screen.getByText('Sair')).toBeTruthy();
  });
});
