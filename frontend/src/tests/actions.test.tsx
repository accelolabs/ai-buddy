import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { ActionCard } from '@/widgets/ai-buddy-chat/ui/ActionCard';
import { changeUser, useSession } from '@/shared/api/session';
import { ApiError, queryClient, request } from '@/shared/api';

beforeEach(() => changeUser('intern'));
afterEach(() => {
  cleanup();
  changeUser(null);
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});
const renderCard = (kind: 'ticket' | 'task' | 'complete' = 'ticket') =>
  render(
    <MemoryRouter>
      <ActionCard
        kind={kind}
        draft={
          kind === 'ticket'
            ? { title: 'Тема', description: 'Описание' }
            : {
                id: 'task-1',
                description: 'Задача',
                deadline: '2027-01-01T12:00:00+03:00',
                user_id: 'intern',
                status: 'pending',
                created_at: '',
              }
        }
      />
    </MemoryRouter>,
  );

describe('explicit action confirmation', () => {
  it('edits a ticket, locks repeated clicks and invalidates only after success', async () => {
    let resolve!: (response: Response) => void;
    const fetch = vi.fn<typeof globalThis.fetch>(
      () =>
        new Promise<Response>((r) => {
          resolve = r;
        }),
    );
    vi.stubGlobal('fetch', fetch);
    const invalidate = vi.spyOn(queryClient, 'invalidateQueries');
    renderCard();
    expect(fetch).not.toHaveBeenCalled();
    fireEvent.change(screen.getByLabelText('Тема'), { target: { value: 'Новая тема' } });
    const button = screen.getByRole('button', { name: 'Подтвердить создание' });
    fireEvent.click(button);
    fireEvent.click(button);
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(JSON.parse(fetch.mock.calls[0][1]?.body as string)).toEqual({
      title: 'Новая тема',
      description: 'Описание',
    });
    expect(invalidate).not.toHaveBeenCalled();
    expect(screen.queryByText('Запись создана')).not.toBeInTheDocument();
    resolve(
      new Response(JSON.stringify({ id: 'real-id' }), {
        status: 201,
        headers: { 'Content-Type': 'application/json' },
      }),
    );
    await screen.findByRole('status');
    expect(screen.getByRole('link', { name: 'Открыть' })).toHaveAttribute(
      'href',
      '/tickets/real-id',
    );
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ['tickets'] });
  });
  it('shows server text errors and leaves the form retryable', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response('database unavailable', { status: 503 })),
    );
    renderCard();
    fireEvent.click(screen.getByRole('button', { name: 'Подтвердить создание' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('database unavailable');
    expect(screen.queryByRole('status')).not.toBeInTheDocument();
    expect(screen.getByRole('button')).toBeEnabled();
  });
  it('sends RFC3339 deadlines and completes tasks with an empty body', async () => {
    const fetch = vi
      .fn()
      .mockImplementation(() =>
        Promise.resolve(
          new Response('{"id":"t"}', { headers: { 'Content-Type': 'application/json' } }),
        ),
      );
    vi.stubGlobal('fetch', fetch);
    const view = renderCard('task');
    fireEvent.click(screen.getByRole('button', { name: 'Подтвердить создание' }));
    await screen.findByRole('status');
    const payload = JSON.parse(fetch.mock.calls[0][1]?.body as string);
    expect(payload.deadline).toMatch(/T\d\d:\d\d:\d\d\.\d{3}Z$/);
    expect(new Date(payload.deadline).toISOString()).toBe('2027-01-01T09:00:00.000Z');
    view.unmount();
    renderCard('complete');
    expect(fetch).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByRole('button', { name: 'Завершить задачу' }));
    await screen.findByRole('status');
    expect(fetch.mock.calls[1][0]).toBe('/api/tracker/tasks/task-1/complete');
    expect(fetch.mock.calls[1][1].body).toBeUndefined();
  });
  it('resets user, chat and cache on 401 and preserves real error text', async () => {
    queryClient.setQueryData(['private'], 'secret');
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response('select a user', { status: 401 })),
    );
    await expect(request('/api/tickets')).rejects.toEqual(new ApiError(401, 'select a user'));
    expect(useSession.getState().userId).toBeNull();
    expect(document.cookie).not.toContain('demo_user=intern');
    expect(queryClient.getQueryData(['private'])).toBeUndefined();
  });
  it('does not apply stale responses or stale authorization errors after changing users', async () => {
    let resolve!: (response: Response) => void;
    vi.stubGlobal(
      'fetch',
      vi.fn(
        () =>
          new Promise<Response>((r) => {
            resolve = r;
          }),
      ),
    );
    const pending = request('/api/tickets');
    changeUser('new-user');
    resolve(new Response('select a user', { status: 401 }));
    await expect(pending).rejects.toThrow();
    await waitFor(() => expect(useSession.getState().userId).toBe('new-user'));
  });
});
