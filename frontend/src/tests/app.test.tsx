import { expect, it } from 'vitest';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { App } from '@/app/App';
import { changeUser, useSession } from '@/shared/api/session';
import { mockApi, json, intern, mentor, secondIntern, task } from './support';

function open(path = '/', id: string | null = intern.id) {
  changeUser(id);
  window.history.replaceState({}, '', path);
  return render(<App />);
}
it('requires login and enters the intern chat after selecting a real user', async () => {
  mockApi();
  open('/tickets', null);
  await screen.findByRole('heading', { name: 'Вход в AI–buddy' });
  fireEvent.click(screen.getByRole('button', { name: /Анна Стажёр/ }));
  fireEvent.mouseDown(document.body);
  expect(screen.queryByRole('listbox')).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: /Анна Стажёр/ }));
  fireEvent.click(screen.getByRole('option', { name: /Иван Стажёр/ }));
  fireEvent.click(screen.getByRole('button', { name: 'Войти' }));
  await screen.findByRole('heading', { name: 'Привет! Я — твой AI-buddy.' });
  expect(useSession.getState().userId).toBe(secondIntern.id);
  expect(screen.getByRole('navigation', { name: 'Приложения' })).toHaveTextContent(
    'ЧатЗадачиТикетыЗнания',
  );
});
it('enters the mentor route, changes interns and keeps tasks read-only', async () => {
  const api = mockApi();
  open('/login', null);
  fireEvent.click(await screen.findByRole('button', { name: /Анна Стажёр/ }));
  fireEvent.click(screen.getByRole('option', { name: /Мария Наставник/ }));
  fireEvent.click(screen.getByRole('button', { name: 'Войти' }));
  await screen.findByRole('heading', { name: 'Мои стажеры' });
  await screen.findByRole('heading', { name: task.description });
  fireEvent.click(screen.getByRole('button', { name: secondIntern.name }));
  await waitFor(() =>
    expect(
      api.mock.calls.some(([url]) => String(url).includes(`/interns/${secondIntern.id}/tasks`)),
    ).toBe(true),
  );
  expect(screen.queryByRole('button', { name: 'Завершить задачу' })).not.toBeInTheDocument();
  expect(screen.queryByRole('link', { name: 'Чат' })).not.toBeInTheDocument();
});
it.each(['/chat', '/dashboard'])('redirects mentor entry %s to assigned interns', async (path) => {
  mockApi();
  open(path, mentor.id);
  await screen.findByRole('heading', { name: 'Мои стажеры' });
  expect(window.location.pathname).toBe('/dashboard');
});
it('navigates all sections and logs out through the profile menu', async () => {
  const user = userEvent.setup();
  mockApi();
  open();
  const nav = await screen.findByRole('navigation');
  await user.click(within(nav).getByRole('link', { name: 'Знания' }));
  await user.click(await screen.findByRole('link', { name: /Инструкция/ }));
  await screen.findByText('Текст статьи');
  await user.click(screen.getByRole('link', { name: 'Все статьи' }));
  await screen.findByRole('link', { name: /Инструкция/ });
  await user.click(within(nav).getByRole('link', { name: 'Тикеты' }));
  await screen.findByRole('heading', { name: 'Нужен доступ', level: 2 });
  await user.click(within(nav).getByRole('link', { name: 'Задачи' }));
  await screen.findByRole('heading', { name: 'Мой маршрут' });
  await user.click(within(nav).getByRole('link', { name: 'Чат' }));
  await screen.findByRole('heading', { name: 'Привет! Я — твой AI-buddy.' });
  const profile = screen.getByRole('button', { name: `Профиль: ${intern.name}` });
  await user.click(profile);
  await screen.findByRole('menu');
  await user.keyboard('{Escape}');
  await waitFor(() => expect(screen.queryByRole('menu')).not.toBeInTheDocument());
  await user.click(profile);
  await user.click(await screen.findByRole('menuitem', { name: 'Выйти' }));
  await screen.findByRole('heading', { name: 'Вход в AI–buddy' });
  expect(useSession.getState().userId).toBeNull();
});
it('shows login errors, retries the user list and handles an empty list', async () => {
  let fail = true;
  mockApi((path) =>
    path === '/api/users'
      ? fail
        ? new Response('users unavailable', { status: 503 })
        : json([])
      : undefined,
  );
  open('/login', null);
  expect(await screen.findByRole('alert')).toHaveTextContent('users unavailable');
  fail = false;
  fireEvent.click(screen.getByRole('button', { name: 'Повторить' }));
  await screen.findByText('Нет пользователей');
});
it('shows protected loading/error states and redirects unknown users to login', async () => {
  let fail = true;
  mockApi((path) =>
    path === '/api/users'
      ? fail
        ? new Response('not ready', { status: 503 })
        : json([])
      : undefined,
  );
  open();
  expect(screen.getByRole('status')).toHaveTextContent('Загрузка');
  await screen.findByRole('alert');
  fail = false;
  fireEvent.click(screen.getByRole('button', { name: 'Повторить' }));
  await waitFor(() => expect(window.location.pathname).toBe('/login'));
});
it('recovers task list and completion failures, updating progress only after the API succeeds', async () => {
  let listFail = true,
    completeFail = true,
    completed = false;
  const api = mockApi((path, options) => {
    if (path === '/api/tracker/tasks')
      return listFail
        ? new Response('tasks unavailable', { status: 503 })
        : json([{ ...task, status: completed ? 'completed' : 'pending' }]);
    if (options.method === 'POST') {
      expect(options.body).toBeUndefined();
      if (completeFail) return new Response('cannot complete', { status: 503 });
      completed = true;
      return json({ ...task, status: 'completed' });
    }
  });
  open('/route');
  await screen.findByText('tasks unavailable');
  listFail = false;
  fireEvent.click(screen.getByRole('button', { name: 'Повторить' }));
  const complete = await screen.findByRole('button', { name: 'Завершить задачу' });
  fireEvent.click(complete);
  await screen.findByText('cannot complete');
  expect(completed).toBe(false);
  completeFail = false;
  fireEvent.click(screen.getByRole('button', { name: 'Завершить задачу' }));
  await screen.findByText('01 · Выполнено');
  expect(screen.queryByRole('button', { name: 'Завершить задачу' })).not.toBeInTheDocument();
  expect(api.mock.calls.some(([path]) => String(path).endsWith('/task-1/complete'))).toBe(true);
});
it.each([false, true])('shows empty task/intern lists for mentor=%s', async (isMentor) => {
  mockApi((path) =>
    path.includes('/tasks') || path === '/api/users/interns' ? json([]) : undefined,
  );
  open('/route', isMentor ? mentor.id : intern.id);
  await screen.findByText(isMentor ? 'Нет назначенных стажёров' : 'Задач пока нет');
});
it('retries a failed interns request', async () => {
  let fail = true;
  mockApi((path) =>
    path === '/api/users/interns'
      ? fail
        ? new Response('interns unavailable', { status: 503 })
        : json([])
      : undefined,
  );
  open('/route', mentor.id);
  await screen.findByText('interns unavailable');
  fail = false;
  fireEvent.click(screen.getByRole('button', { name: 'Повторить' }));
  await screen.findByText('Нет назначенных стажёров');
});
it.each(['/unknown', '/chat'])('opens intern chat at %s', async (path) => {
  mockApi();
  open(path);
  await screen.findByRole('heading', { name: 'Привет! Я — твой AI-buddy.' });
});
it('selects a ticket via its actual detail endpoint and rejects unknown IDs', async () => {
  const api = mockApi((path) =>
    path === '/api/tickets/missing' ? new Response('not found', { status: 404 }) : undefined,
  );
  open('/tickets');
  const link = await screen.findByRole('link', { name: /Нужен доступ/ });
  fireEvent.click(link);
  await waitFor(() =>
    expect(api.mock.calls.some(([path]) => path === '/api/tickets/ticket-1')).toBe(true),
  );
  window.history.pushState({}, '', '/tickets/missing');
  fireEvent.popState(window);
  await screen.findByText('Тикет не найден');
  expect(screen.queryByRole('heading', { level: 2, name: 'Нужен доступ' })).not.toBeInTheDocument();
});
it('retries ticket list/detail failures and supports an empty ticket list', async () => {
  let listFail = true,
    detailFail = true;
  mockApi((path) =>
    path === '/api/tickets'
      ? listFail
        ? new Response('list failed', { status: 503 })
        : json([])
      : path === '/api/tickets/ticket-1'
        ? detailFail
          ? new Response('detail failed', { status: 503 })
          : json({ ...task, title: 'Тикет', description: 'Текст', user_id: intern.id })
        : undefined,
  );
  open('/tickets/ticket-1');
  await screen.findByText('list failed');
  await screen.findByText('detail failed');
  listFail = false;
  detailFail = false;
  screen.getAllByRole('button', { name: 'Повторить' }).forEach((button) => fireEvent.click(button));
  await screen.findByText('Тикетов пока нет');
  await screen.findByRole('heading', { name: 'Тикет', level: 2 });
});
it('renders all ticket circuit variants when several tickets are available', async () => {
  mockApi((path) =>
    path === '/api/tickets'
      ? json(
          Array.from({ length: 4 }, (_, i) => ({
            id: `t-${i}`,
            user_id: intern.id,
            title: `Тикет ${i}`,
            description: 'Описание',
            created_at: task.created_at,
          })),
        )
      : undefined,
  );
  open('/tickets');
  await screen.findByRole('heading', { level: 2, name: 'Тикет 0' });
  expect(screen.getAllByRole('link', { name: /Тикет \d/ })).toHaveLength(4);
});
