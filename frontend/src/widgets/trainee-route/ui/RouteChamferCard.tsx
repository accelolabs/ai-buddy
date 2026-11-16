import { Check, Loader2 } from 'lucide-react';
import type { Task } from '@/shared/api/contracts';
import { taskStatus, calendarDate } from '@/shared/api/contracts';
export function RouteChamferCard({
  task,
  order,
  onComplete,
  pending,
}: {
  task: Task;
  order?: number;
  onComplete?: () => void;
  pending: boolean;
}) {
  const color =
    task.status === 'completed' ? '#10B981' : task.status === 'overdue' ? '#F43F5E' : '#CBD5E1';
  return (
    <div className="relative w-full min-h-[94px]">
      <svg
        viewBox="0 0 810 94"
        className="pointer-events-none absolute inset-0 h-full w-full"
        preserveAspectRatio="none"
      >
        <path
          d="M13 1H809V81L797 93H1V13L13 1Z"
          fill="white"
          stroke={color}
          strokeWidth="1.5"
          vectorEffect="non-scaling-stroke"
        />
      </svg>
      <div className="relative flex min-h-[94px] flex-wrap items-center justify-between gap-4 px-5 py-4 sm:px-7">
        <div className="min-w-0 flex-1">
          <p
            className="text-xs font-semibold"
            style={{ color: task.status === 'pending' ? '#0284C7' : color }}
          >
            {order !== undefined && <>{String(order).padStart(2, '0')} · </>}
            {taskStatus(task.status)}
          </p>
          <h3 className="mt-1 break-words font-bold text-slate-900">{task.description}</h3>
          <p className="text-xs text-slate-500">Срок: {calendarDate(task.deadline)}</p>
        </div>
        {onComplete && task.status !== 'completed' && (
          <button
            type="button"
            aria-label="Завершить задачу"
            title={pending ? 'Сохранение…' : 'Завершить задачу'}
            onClick={onComplete}
            disabled={pending}
            className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-slate-50 text-slate-400 transition-colors hover:border-sky-300 hover:bg-sky-50 hover:text-sky-600 disabled:opacity-50"
          >
            {pending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Check className="h-4 w-4" />}
          </button>
        )}
      </div>
    </div>
  );
}
