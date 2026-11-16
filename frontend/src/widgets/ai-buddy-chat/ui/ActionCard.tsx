import { useRef, useState } from 'react';
import { Link } from 'react-router';
import { request, queryClient } from '@/shared/api';
import { sessionSignal } from '@/shared/api/session';
import type { Task, TaskDraft, Ticket, TicketDraft } from '@/shared/api/contracts';
import { dateTime } from '@/shared/api/contracts';

export function ActionCard({
  kind,
  draft,
}: {
  kind: 'ticket' | 'task' | 'complete';
  draft: TicketDraft | TaskDraft | Task;
}) {
  const [title, setTitle] = useState('title' in draft ? draft.title : '');
  const [description, setDescription] = useState(draft.description);
  const [deadline, setDeadline] = useState('deadline' in draft ? localDate(draft.deadline) : '');
  const [pending, setPending] = useState(false);
  const [result, setResult] = useState<Task | Ticket | null>(null);
  const [error, setError] = useState('');
  const lock = useRef(false);
  const signal = sessionSignal();
  const complete = kind === 'complete';
  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (lock.current || result || signal.aborted) return;
    lock.current = true;
    setPending(true);
    setError('');
    try {
      const path =
        kind === 'ticket'
          ? '/api/tickets'
          : complete
            ? `/api/tracker/tasks/${encodeURIComponent((draft as Task).id)}/complete`
            : '/api/tracker/tasks';
      const body = complete
        ? undefined
        : JSON.stringify(
            kind === 'ticket'
              ? { title: title.trim(), description: description.trim() }
              : { description: description.trim(), deadline: new Date(deadline).toISOString() },
          );
      const value = await request<Task | Ticket>(path, { method: 'POST', body, signal });
      if (signal.aborted) return;
      setResult(value);
      await queryClient.invalidateQueries({ queryKey: [kind === 'ticket' ? 'tickets' : 'tasks'] });
    } catch (err) {
      if (!signal.aborted) setError(err instanceof Error ? err.message : String(err));
    } finally {
      lock.current = false;
      if (!signal.aborted) setPending(false);
    }
  };
  return (
    <form
      onSubmit={submit}
      className="rounded-2xl border border-sky-200 bg-white p-5 shadow-xs space-y-3"
    >
      <h3 className="font-bold text-slate-900">
        {kind === 'ticket' ? 'Обращение' : complete ? 'Завершение задачи' : 'Новая задача'}
      </h3>
      <fieldset disabled={pending || !!result} className="space-y-3">
        {kind === 'ticket' && (
          <label className="block text-sm">
            Тема
            <input
              required
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="mt-1 w-full rounded-lg border p-2"
            />
          </label>
        )}
        <label className="block text-sm">
          Описание
          <textarea
            required
            readOnly={complete}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="mt-1 w-full rounded-lg border p-2"
          />
        </label>
        {kind === 'task' && (
          <label className="block text-sm">
            Срок (местное время)
            <input
              required
              type="datetime-local"
              value={deadline}
              onChange={(e) => setDeadline(e.target.value)}
              className="mt-1 w-full rounded-lg border p-2"
            />
          </label>
        )}
        {complete && (
          <p className="text-sm text-slate-500">Срок: {dateTime((draft as Task).deadline)}</p>
        )}
        {!result && (
          <button
            disabled={complete && (draft as Task).status === 'completed'}
            className="rounded-xl bg-sky-500 px-4 py-2 font-semibold text-white disabled:opacity-50"
          >
            {pending ? 'Сохранение…' : complete ? 'Завершить задачу' : 'Подтвердить создание'}
          </button>
        )}
      </fieldset>
      {error && (
        <p role="alert" className="text-red-700">
          {error}
        </p>
      )}
      {result && (
        <p role="status" className="text-emerald-700">
          {complete ? 'Задача завершена' : 'Запись создана'} ·{' '}
          <Link
            className="underline"
            to={kind === 'ticket' ? `/tickets/${encodeURIComponent(result.id)}` : '/route'}
          >
            Открыть
          </Link>
        </p>
      )}
    </form>
  );
}
function localDate(value: string) {
  const date = new Date(value);
  return new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16);
}
