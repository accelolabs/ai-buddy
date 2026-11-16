import { useRef, useState } from 'react';
import { Link } from 'react-router';
import { AlertCircle, Check, ExternalLink, FileText, Loader2, Pencil } from 'lucide-react';
import { request, queryClient } from '@/shared/api';
import { sessionSignal } from '@/shared/api/session';
import type { Task, TaskDraft, Ticket, TicketDraft } from '@/shared/api/contracts';
import { RouteChamferCard } from '@/widgets/trainee-route';

export function ActionCard({
  kind,
  draft,
  order,
}: {
  kind: 'ticket' | 'task' | 'complete';
  order?: number;
  draft: TicketDraft | TaskDraft | Task;
}) {
  const titleInput = useRef<HTMLInputElement>(null);
  const descriptionInput = useRef<HTMLTextAreaElement>(null);
  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState('title' in draft ? draft.title : '');
  const [description, setDescription] = useState(draft.description);
  const [deadline, setDeadline] = useState('deadline' in draft ? draft.deadline : '');
  const [pending, setPending] = useState(false);
  const [result, setResult] = useState<Task | Ticket | null>(null);
  const [error, setError] = useState('');
  const lock = useRef(false);
  const signal = sessionSignal();
  const complete = kind === 'complete';
  const submit = async (event?: React.FormEvent) => {
    event?.preventDefault();
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
              : { description: description.trim(), deadline },
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
  if (complete)
    return (
      <div className="space-y-3">
        <RouteChamferCard
          task={{ ...(draft as Task), ...(result as Task | null) }}
          order={order}
          pending={pending}
          onComplete={!result ? () => void submit() : undefined}
        />
        {error && (
          <p role="alert" className="text-red-700">
            {error}
          </p>
        )}
        {result && (
          <p role="status" className="text-sm text-emerald-700">
            Задача завершена
          </p>
        )}
      </div>
    );
  if (kind === 'ticket') {
    const recipient = (draft as TicketDraft).recipient;
    if (result)
      return (
        <div
          role="status"
          className="space-y-3 rounded-2xl border border-emerald-200 bg-emerald-50/70 p-4 text-emerald-950 shadow-xs"
        >
          <div className="flex items-start gap-3">
            <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-emerald-100 text-emerald-600">
              <Check className="h-4 w-4" />
            </span>
            <p className="min-w-0 break-words pt-1 text-sm">
              <strong>Тикет #{result.id}</strong> «{(result as Ticket).title || title}» создан.
            </p>
          </div>
          <Link
            className="inline-flex max-w-full items-center gap-1.5 rounded-full border border-emerald-100 bg-white px-3 py-1.5 text-xs font-semibold text-sky-600 hover:bg-sky-50"
            to={`/tickets/${encodeURIComponent(result.id)}`}
          >
            <span className="min-w-0 break-words">Перейти к тикету #{result.id}</span>
            <ExternalLink className="h-3.5 w-3.5 shrink-0" />
          </Link>
        </div>
      );
    return (
      <form
        onSubmit={submit}
        className="rounded-2xl border border-sky-200 bg-slate-50/80 p-4 shadow-xs sm:p-5"
      >
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3 border-b border-slate-200/70 pb-3">
          <span className="inline-flex items-center gap-1.5 rounded-full bg-sky-100 px-2.5 py-1 text-xs font-bold text-sky-600">
            <FileText className="h-3.5 w-3.5" />
            Черновик тикета
          </span>
          {recipient && (
            <div className="flex min-w-0 items-center gap-2 text-xs">
              <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-sky-200 text-[10px] font-bold text-sky-800">
                {recipient.name
                  .trim()
                  .split(/\s+/)
                  .slice(0, 2)
                  .map((name) => name[0])
                  .join('')}
              </span>
              <p className="min-w-0 break-words">
                <span className="text-slate-500">Получатель: </span>
                <strong className="text-slate-700">{recipient.name}</strong>
              </p>
            </div>
          )}
        </div>
        <fieldset disabled={pending} className="space-y-4">
          <label className="block text-[11px] font-semibold uppercase tracking-wide text-slate-500">
            Тема обращения
            <input
              ref={titleInput}
              required
              readOnly={!editing}
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              className="mt-1.5 w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm font-semibold normal-case tracking-normal text-slate-800 outline-none focus:border-sky-300 focus:ring-2 focus:ring-sky-100"
            />
          </label>
          <label className="block text-[11px] font-semibold uppercase tracking-wide text-slate-500">
            Текст сообщения
            <textarea
              required
              readOnly={!editing}
              rows={6}
              value={description}
              onChange={(event) => setDescription(event.target.value)}
              className="mt-1.5 w-full resize-y rounded-xl border border-slate-200 bg-white px-3 py-3 text-sm font-normal normal-case leading-relaxed tracking-normal text-slate-700 outline-none focus:border-sky-300 focus:ring-2 focus:ring-sky-100"
            />
          </label>
          {error && (
            <div
              role="alert"
              className="flex items-start gap-2 rounded-xl border border-red-200 bg-red-50 p-3 text-sm text-red-800"
            >
              <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
              <div>
                <p className="font-semibold">Не удалось создать тикет</p>
                <p className="mt-1 break-words">{error}</p>
              </div>
            </div>
          )}
          <div className="flex flex-wrap items-center justify-end gap-2">
            <p className="mr-auto text-xs text-slate-500">Проверьте тему и текст обращения.</p>
            <button
              type="button"
              onClick={() => {
                setEditing(!editing);
                if (!editing) requestAnimationFrame(() => titleInput.current?.focus());
              }}
              className="inline-flex items-center justify-center gap-1.5 rounded-full border border-slate-200 bg-white px-3 py-2 text-xs font-semibold text-slate-600 hover:bg-slate-100 disabled:opacity-50"
            >
              <Pencil className="h-3.5 w-3.5" />
              {editing ? 'Готово' : 'Изменить'}
            </button>
            <button
              type="submit"
              disabled={pending || !title.trim() || !description.trim()}
              className="inline-flex items-center justify-center gap-1.5 rounded-full bg-sky-500 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-sky-600 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {pending ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <Check className="h-3.5 w-3.5" />
              )}
              {pending ? 'Создание…' : error ? 'Повторить отправку' : 'Подтвердить и отправить'}
            </button>
          </div>
        </fieldset>
      </form>
    );
  }
  if (result)
    return (
      <div
        role="status"
        className="space-y-3 rounded-2xl border border-emerald-200 bg-emerald-50/70 p-4 text-emerald-950 shadow-xs"
      >
        <div className="flex items-start gap-3">
          <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-emerald-100 text-emerald-600">
            <Check className="h-4 w-4" />
          </span>
          <p className="min-w-0 break-words pt-1 text-sm">
            Задача «{(result as Task).description || description}» создана.
          </p>
        </div>
        <Link
          className="inline-flex items-center gap-1.5 rounded-full border border-emerald-100 bg-white px-3 py-1.5 text-xs font-semibold text-sky-600 hover:bg-sky-50"
          to="/route"
        >
          Открыть задачи
          <ExternalLink className="h-3.5 w-3.5" />
        </Link>
      </div>
    );
  return (
    <form
      onSubmit={submit}
      className="rounded-2xl border border-sky-200 bg-slate-50/80 p-4 shadow-xs sm:p-5"
    >
      <div className="mb-4 flex items-center border-b border-slate-200/70 pb-3">
        <span className="inline-flex items-center gap-1.5 rounded-full bg-sky-100 px-2.5 py-1 text-xs font-bold text-sky-600">
          <FileText className="h-3.5 w-3.5" />
          Черновик задачи
        </span>
      </div>
      <fieldset disabled={pending} className="space-y-4">
        <label className="block text-[11px] font-semibold uppercase tracking-wide text-slate-500">
          Описание задачи
          <textarea
            ref={descriptionInput}
            required
            readOnly={!editing}
            rows={4}
            value={description}
            onChange={(event) => setDescription(event.target.value)}
            className="mt-1.5 w-full resize-y rounded-xl border border-slate-200 bg-white px-3 py-3 text-sm font-normal normal-case leading-relaxed tracking-normal text-slate-700 outline-none focus:border-sky-300 focus:ring-2 focus:ring-sky-100"
          />
        </label>
        <label className="block text-[11px] font-semibold uppercase tracking-wide text-slate-500">
          Срок
          <input
            required
            readOnly={!editing}
            type="date"
            value={deadline}
            onChange={(event) => setDeadline(event.target.value)}
            className="mt-1.5 w-full min-w-0 rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm font-normal normal-case tracking-normal text-slate-800 outline-none focus:border-sky-300 focus:ring-2 focus:ring-sky-100"
          />
        </label>
        {error && (
          <div
            role="alert"
            className="flex items-start gap-2 rounded-xl border border-red-200 bg-red-50 p-3 text-sm text-red-800"
          >
            <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
            <div>
              <p className="font-semibold">Не удалось создать задачу</p>
              <p className="mt-1 break-words">{error}</p>
            </div>
          </div>
        )}
        <div className="flex flex-wrap items-center justify-end gap-2">
          <p className="mr-auto text-xs text-slate-500">Проверьте описание и срок задачи.</p>
          <button
            type="button"
            onClick={() => {
              setEditing(!editing);
              if (!editing) requestAnimationFrame(() => descriptionInput.current?.focus());
            }}
            className="inline-flex items-center justify-center gap-1.5 rounded-full border border-slate-200 bg-white px-3 py-2 text-xs font-semibold text-slate-600 hover:bg-slate-100 disabled:opacity-50"
          >
            <Pencil className="h-3.5 w-3.5" />
            {editing ? 'Готово' : 'Изменить'}
          </button>
          <button
            type="submit"
            disabled={pending || !description.trim() || !deadline}
            className="inline-flex items-center justify-center gap-1.5 rounded-full bg-sky-500 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-sky-600 disabled:cursor-not-allowed disabled:opacity-50"
          >
            {pending ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Check className="h-3.5 w-3.5" />
            )}
            {pending ? 'Создание…' : error ? 'Повторить создание' : 'Подтвердить создание'}
          </button>
        </div>
      </fieldset>
    </form>
  );
}
