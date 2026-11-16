import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { ChatMessageItem } from '@/widgets/ai-buddy-chat/ui/ChatMessageItem';
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
            ? {
                title: 'Тема',
                description: 'Описание',
                recipient: { id: 'mentor', name: 'Мария Соколова' },
              }
            : {
                id: 'task-1',
                description: 'Задача',
                deadline: '2027-01-01',
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
    expect(screen.getByText('Мария Соколова')).toBeInTheDocument();
    expect(screen.getByLabelText('Тема обращения')).toHaveAttribute('readonly');
    fireEvent.click(screen.getByRole('button', { name: 'Изменить' }));
    fireEvent.change(screen.getByLabelText('Тема обращения'), { target: { value: 'Новая тема' } });
    const button = screen.getByRole('button', { name: 'Подтвердить и отправить' });
    fireEvent.click(button);
    fireEvent.click(button);
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(JSON.parse(fetch.mock.calls[0][1]?.body as string)).toEqual({
      title: 'Новая тема',
      description: 'Описание',
    });
    expect(invalidate).not.toHaveBeenCalled();
    expect(screen.queryByRole('status')).not.toBeInTheDocument();
    resolve(
      new Response(JSON.stringify({ id: 'real-id' }), {
        status: 201,
        headers: { 'Content-Type': 'application/json' },
      }),
    );
    await screen.findByRole('status');
    expect(screen.getByRole('status')).toHaveTextContent('Тикет #real-id «Новая тема» создан.');
    expect(screen.queryByText('Черновик тикета')).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Перейти к тикету #real-id' })).toHaveAttribute(
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
    fireEvent.click(screen.getByRole('button', { name: 'Подтвердить и отправить' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('database unavailable');
    expect(screen.queryByRole('status')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Повторить отправку' })).toBeEnabled();
    expect(screen.getByLabelText('Тема обращения')).toHaveValue('Тема');
    expect(screen.getByLabelText('Текст сообщения')).toHaveValue('Описание');
  });
  it('sends calendar deadlines and completes tasks with an empty body', async () => {
    const fetch = vi
      .fn()
      .mockImplementation(() =>
        Promise.resolve(
          new Response('{"id":"t"}', { headers: { 'Content-Type': 'application/json' } }),
        ),
      );
    vi.stubGlobal('fetch', fetch);
    const view = renderCard('task');
    expect(screen.getByText('Черновик задачи')).toBeInTheDocument();
    expect(screen.getByLabelText('Описание задачи')).toHaveAttribute('readonly');
    fireEvent.click(screen.getByRole('button', { name: 'Изменить' }));
    fireEvent.change(screen.getByLabelText('Описание задачи'), {
      target: { value: 'Обновлённая задача' },
    });
    const deadlineInput = screen.getByLabelText('Срок') as HTMLInputElement;
    fireEvent.change(deadlineInput, {
      target: { value: deadlineInput.value.replace('2027-01-01', '2027-01-02') },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Готово' }));
    fireEvent.click(screen.getByRole('button', { name: 'Подтвердить создание' }));
    await screen.findByRole('status');
    const payload = JSON.parse(fetch.mock.calls[0][1]?.body as string);
    expect(deadlineInput).toHaveAttribute('type', 'date');
    expect(payload.deadline).toBe('2027-01-02');
    expect(payload.description).toBe('Обновлённая задача');
    view.unmount();
    renderCard('complete');
    expect(screen.getByText('Срок: 01.01.2027')).toBeInTheDocument();
    expect(fetch).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByRole('button', { name: 'Завершить задачу' }));
    await screen.findByRole('status');
    expect(fetch.mock.calls[1][0]).toBe('/api/tracker/tasks/task-1/complete');
    expect(fetch.mock.calls[1][1].body).toBeUndefined();
  });
  it('shows only the selected task card for completion and updates it after confirmation', async () => {
    const selected = {
      id: 'selected',
      user_id: 'intern',
      description: 'Выбранная задача',
      status: 'pending' as const,
      deadline: '2027-01-01',
      created_at: '',
    };
    const fetch = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ ...selected, status: 'completed' }), {
        headers: { 'Content-Type': 'application/json' },
      }),
    );
    vi.stubGlobal('fetch', fetch);
    render(
      <MemoryRouter>
        <ChatMessageItem
          busy={false}
          onRetry={() => {}}
          message={{
            id: 'answer',
            role: 'assistant',
            content: 'Подтвердите завершение',
            status: 'completed',
            stages: [],
            tasks: [{ ...selected, id: 'other', description: 'Другая задача' }, selected],
            task_completion: selected,
          }}
        />
      </MemoryRouter>,
    );
    expect(screen.getByRole('heading', { name: 'Выбранная задача' })).toBeInTheDocument();
    expect(screen.getByText('02 · Впереди')).toBeInTheDocument();
    expect(screen.queryByText('Другая задача')).not.toBeInTheDocument();
    expect(screen.queryByRole('textbox')).not.toBeInTheDocument();
    expect(fetch).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: 'Завершить задачу' }));
    await screen.findByRole('status');
    expect(screen.getByText('02 · Выполнено')).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Завершить задачу' })).not.toBeInTheDocument();
    expect(fetch.mock.calls[0][0]).toBe('/api/tracker/tasks/selected/complete');
    expect(fetch.mock.calls[0][1].body).toBeUndefined();
  });
  it('keeps the completion card retryable when the server fails', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response('failed to complete', { status: 503 })),
    );
    renderCard('complete');
    fireEvent.click(screen.getByRole('button', { name: 'Завершить задачу' }));
    expect(await screen.findByRole('alert')).toHaveTextContent('failed to complete');
    expect(screen.queryByRole('status')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Завершить задачу' })).toBeEnabled();
    expect(screen.getByText('Впереди')).toBeInTheDocument();
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
