import { useState } from 'react';
import { Link } from 'react-router';
import { Copy, Check, RotateCcw } from 'lucide-react';
import type { ChatMessage } from '@/entities/chat';
import { BuddyAvatar } from './BuddyAvatar';
import { ExecutionStages } from './ExecutionStages';
import { ActionCard } from './ActionCard';
import { RouteChamferCard, RouteWireSegment } from '@/widgets/trainee-route';

export function ChatMessageItem({
  message,
  onRetry,
  busy,
}: {
  message: ChatMessage;
  onRetry: (id: string) => void;
  busy: boolean;
}) {
  const [copied, setCopied] = useState(false);
  const [copyError, setCopyError] = useState('');
  if (message.role === 'user')
    return (
      <div className="flex justify-end py-2">
        <div className="max-w-[85%] whitespace-pre-wrap rounded-[20px] rounded-tr-sm bg-[#F1F5F9] px-5 py-3 text-[15px]">
          {message.content}
        </div>
      </div>
    );
  const completionIndex =
    message.tasks?.findIndex((task) => task.id === message.task_completion?.id) ?? -1;
  return (
    <div className="flex gap-3 py-3">
      <BuddyAvatar className="h-8 w-8 shrink-0" />
      <div className="min-w-0 flex-1 space-y-3.5">
        <ExecutionStages
          stages={message.stages}
          showSkeleton={message.status === 'in_progress' && !message.content}
          isCompleted={message.status === 'completed'}
        />
        <div className="whitespace-pre-wrap break-words text-[15px] leading-relaxed text-[#0F172A]">
          {message.content}
        </div>
        {message.source && (
          <Link
            className="block rounded-2xl border border-sky-200 bg-sky-50/50 p-4"
            to={`/wiki/${encodeURIComponent(message.source.document_id)}#${encodeURIComponent(message.source.anchor)}`}
          >
            <strong>{message.source.title}</strong>
            <blockquote className="mt-2 text-sm text-slate-600">{message.source.quote}</blockquote>
            <span className="text-sm text-sky-700">Читать статью →</span>
          </Link>
        )}
        {message.ticket && <ActionCard kind="ticket" draft={message.ticket} />}
        {message.task && <ActionCard kind="task" draft={message.task} />}
        {message.tasks && !message.task_completion && (
          <div className="rounded-2xl border border-slate-200 bg-white p-4 sm:p-5">
            <h3 className="font-bold">Задачи</h3>
            {message.tasks.length ? (
              message.tasks.map((task, index) => (
                <div key={task.id}>
                  <RouteWireSegment variant={task.status === 'completed' ? 'green' : 'gray'} />
                  <RouteChamferCard task={task} order={index + 1} pending={false} />
                </div>
              ))
            ) : (
              <p>Задач пока нет</p>
            )}
          </div>
        )}
        {message.task_completion && (
          <ActionCard
            kind="complete"
            draft={message.task_completion}
            order={completionIndex >= 0 ? completionIndex + 1 : undefined}
          />
        )}
        {message.error && (
          <p role="alert" className="rounded-xl bg-red-50 p-4 text-red-800">
            {message.error}
          </p>
        )}
        {message.status !== 'in_progress' && (
          <div className="flex gap-3 text-slate-500">
            <button
              aria-label="Копировать"
              onClick={async () => {
                try {
                  await navigator.clipboard.writeText(message.content);
                  setCopied(true);
                } catch {
                  setCopyError('Не удалось скопировать текст');
                }
              }}
            >
              {copied ? <Check size={16} /> : <Copy size={16} />}
            </button>
            <button disabled={busy} title="Повторить ответ" onClick={() => onRetry(message.id)}>
              <RotateCcw size={16} />
            </button>
          </div>
        )}
        {copyError && <p role="alert">{copyError}</p>}
      </div>
    </div>
  );
}
