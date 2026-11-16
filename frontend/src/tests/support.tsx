import { afterEach, beforeEach, vi } from 'vitest';
import { cleanup, render } from '@testing-library/react';
import { QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Route, Routes } from 'react-router';
import type { ReactNode } from 'react';
import { queryClient } from '@/shared/api';
import { changeUser } from '@/shared/api/session';

export const intern = {
  id: 'intern-1',
  name: 'Анна Стажёр',
  role: 'intern' as const,
  team: 'backend',
};
export const secondIntern = { ...intern, id: 'intern-2', name: 'Иван Стажёр' };
export const mentor = {
  id: 'mentor-1',
  name: 'Мария Наставник',
  role: 'mentor' as const,
  team: 'backend',
};
export const task = {
  id: 'task-1',
  user_id: intern.id,
  description: 'Изучить документацию',
  deadline: '2027-01-01',
  status: 'pending' as const,
  created_at: '2026-01-01T12:00:00Z',
};
export const ticket = {
  id: 'ticket-1',
  user_id: intern.id,
  title: 'Нужен доступ',
  description: 'Доступ к проекту',
  created_at: '2026-01-01T12:00:00Z',
};
export const json = (value: unknown, status = 200) =>
  new Response(JSON.stringify(value), { status, headers: { 'Content-Type': 'application/json' } });
export type Handler = (
  path: string,
  options: RequestInit,
) => Response | Promise<Response> | undefined;
export function mockApi(handler: Handler = () => undefined) {
  const fetch = vi.fn(async (input: RequestInfo | URL, options: RequestInit = {}) => {
    const path = new URL(String(input), 'http://localhost').pathname;
    const custom = await handler(path, options);
    if (custom) return custom;
    if (path === '/api/users') return json([intern, secondIntern, mentor]);
    if (path === '/api/users/interns') return json([intern, secondIntern]);
    if (path.includes('/tasks')) return json([task]);
    if (path === '/api/tickets') return json([ticket]);
    if (path === '/api/tickets/ticket-1') return json(ticket);
    if (path === '/api/wiki') return json([{ id: 'doc', title: 'Инструкция', team: 'backend' }]);
    if (path === '/api/wiki/doc') return new Response('# Инструкция\n\n## Раздел\nТекст статьи');
    throw new Error(`Unexpected request: ${path}`);
  });
  vi.stubGlobal('fetch', fetch);
  return fetch;
}
export function mount(element: ReactNode, path = '/', route = '*') {
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path={route} element={element} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  queryClient.setDefaultOptions({ queries: { retry: false }, mutations: { retry: false } });
  changeUser(null);
  window.history.replaceState({}, '', '/');
  Element.prototype.scrollIntoView = vi.fn();
});
afterEach(() => {
  cleanup();
  changeUser(null);
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});
