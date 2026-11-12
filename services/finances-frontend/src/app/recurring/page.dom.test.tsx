/**
 * @vitest-environment happy-dom
 */
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { render, screen, waitFor, cleanup, fireEvent } from '@testing-library/react';
import RecurringPage from './page';

/**
 * The CALL SITE, not the helper.
 *
 * `scheduleForSave` is covered on its own, and that was not enough: a reviewer
 * changed this page to stop passing it the stored row — taking the create
 * branch, which rewrites the schedule to `startOn` — and all 114 tests stayed
 * green. The highest-severity bug in this PR would have come straight back
 * through a refactor nothing was watching. Same lesson as computeNetWorth, one
 * layer out: the helper was right, the wiring was unguarded.
 */

const RECURRING_FIXTURE = {
  id: 'rec-tim',
  profileId: 'p1',
  bankAccountId: 'acc-mp',
  categoryId: 'cat-celular',
  type: 'EXPENSE',
  amount: 99.39,
  currency: 'BRL',
  description: 'Tim Celular',
  // The real shape: a day-of-month qualifier, a start eleven months back, and a
  // next run deliberately advanced past it.
  recurrenceRule: 'FREQ=MONTHLY;BYMONTHDAY=20',
  startOn: '2026-01-20T00:00:00Z',
  nextOccurrence: '2026-12-20T00:00:00Z',
  status: 'ACTIVE',
  createdAt: '',
  updatedAt: '',
};

// vi.hoisted, because vi.mock factories are hoisted above every const in this
// file. Declared plainly, the factory below closes over bindings that do not
// exist yet: the file then throws during collection and reports ZERO tests —
// which the suite summary renders as a green total that simply does not include
// them. It passed when run alone and was broken in the full run.
const { put, patch } = vi.hoisted(() => ({
  put: vi.fn<(path: string, body?: unknown) => Promise<unknown>>(async () => ({})),
  patch: vi.fn<(path: string, body?: unknown) => Promise<unknown>>(async () => ({})),
}));

// Referenced lazily inside the factory, so the hoist is harmless.
vi.mock('@/lib/api', () => ({
  api: {
    get: vi.fn(async (path: string) =>
      path.startsWith('/recurring-transactions') ? { data: [RECURRING_FIXTURE] } : { data: [] }
    ),
    put,
    patch,
    post: vi.fn(async () => ({})),
    delete: vi.fn(async () => ({})),
  },
}));

vi.mock('@/components/layout/AppLayout', () => ({
  default: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  useProfile: () => ({ selectedProfileId: 'p1', selectedProfile: { id: 'p1', name: 'Bruno', type: 'PERSONAL' } }),
}));

vi.mock('@/components/ui/Toast', () => ({
  useToast: () => ({ toast: vi.fn(), confirm: vi.fn(async () => true) }),
}));

beforeEach(() => {
  put.mockClear();
  patch.mockClear();
});
afterEach(cleanup);

const openEditAndSave = async () => {
  render(<RecurringPage />);
  await waitFor(() => expect(screen.getAllByText('Tim Celular').length).toBeGreaterThan(0));

  fireEvent.click(screen.getByTitle('Editar'));

  // Submitting the form itself, not the button: the button is a plain submit
  // and the handler hangs off the <form>.
  const save = await screen.findByRole('button', { name: /salvar/i });
  fireEvent.submit(save.closest('form')!);

  await waitFor(() => expect(put).toHaveBeenCalled());
  return (put.mock.calls[0]?.[1] ?? {}) as Record<string, unknown>;
};

describe('editing a recurrence', () => {
  it('sends the stored next occurrence back, not the start date', async () => {
    const payload = await openEditAndSave();

    expect(payload.nextOccurrence).toBe('2026-12-20');
  });

  it('sends the stored rule back, so the day of month survives', async () => {
    const payload = await openEditAndSave();

    expect(payload.recurrenceRule).toBe('FREQ=MONTHLY;BYMONTHDAY=20');
  });
});

// Pausing with a review date is this page's OTHER full-replace PUT, and it had
// the same defect: the rule rebuilt as FREQ=<freq>, dropping BYMONTHDAY. It was
// fixed in the edit path and missed here — inside the same review cycle — which
// is the argument for pinning the call site rather than only the helper.
describe('pausing a recurrence with a review date', () => {
  it('sends the stored rule and the stored next occurrence back untouched', async () => {
    render(<RecurringPage />);
    await waitFor(() => expect(screen.getAllByText('Tim Celular').length).toBeGreaterThan(0));

    fireEvent.click(screen.getByTitle('Pausar'));
    fireEvent.click(await screen.findByRole('button', { name: '3 meses' }));
    // The row's own "Pausar" icon is still on screen; the modal's confirm is the
    // one inside the dialog that announces "Pausar Transacao".
    const dialog = screen.getByText('Pausar Transacao').closest('div')!.parentElement!;
    const confirm = Array.from(dialog.querySelectorAll('button')).find((b) => b.textContent?.trim() === 'Pausar');
    fireEvent.click(confirm!);

    await waitFor(() => expect(put).toHaveBeenCalled());
    const payload = (put.mock.calls[0]?.[1] ?? {}) as Record<string, unknown>;

    expect(payload.recurrenceRule).toBe('FREQ=MONTHLY;BYMONTHDAY=20');
    expect(payload.nextOccurrence).toBe('2026-12-20');
    expect(payload.status).toBe('PAUSED');
  });
});
