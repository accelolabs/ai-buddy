import { memo } from 'react';
import { Check, AlertCircle } from 'lucide-react';
import type { RouteTask } from '@/entities/trainee-route';

export interface RouteChamferCardProps {
  task: RouteTask;
  onToggle: (taskId: string) => void;
  onRetry: (taskId: string) => void;
}

export const RouteChamferCard = memo(function RouteChamferCard({
  task,
  onToggle,
  onRetry,
}: RouteChamferCardProps) {
  const isCompleted = task.completed;
  const isError = task.status === 'error' || task.hasError;
  const isGlitching = task.isGlitching;

  const orderStr = String(task.order).padStart(2, '0');

  let borderColor = '#CBD5E1';
  let bgColor = '#FFFFFF';
  let statusText = 'Впереди';
  let statusTextColor = '#0284C7';
  let subtitleColor = '#64748B';

  if (isCompleted) {
    borderColor = '#10B981';
    bgColor = '#FFFFFF';
    statusText = 'Выполнено';
    statusTextColor = '#059669';
  } else if (isError) {
    borderColor = '#F43F5E';
    bgColor = '#FFF8F9';
    statusText = 'Просрочено';
    statusTextColor = '#E11D48';
    subtitleColor = '#E11D48';
  }

  return (
    <div
      className={`relative w-full min-h-[94px] transition-transform ${
        isGlitching ? 'animate-glitch-shake' : ''
      }`}
    >
      <svg
        viewBox="0 0 810 94"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className="pointer-events-none absolute inset-0 h-full w-full select-none"
        preserveAspectRatio="none"
      >
        <path
          d="M13 1H809V81L797 93H1V13L13 1Z"
          fill={bgColor}
          stroke={borderColor}
          strokeWidth="1.5"
          vectorEffect="non-scaling-stroke"
        />
      </svg>

      <div className="relative z-10 flex min-h-[94px] items-center justify-between gap-4 px-5 py-3.5 sm:px-7">
        <div className="flex flex-col justify-center">
          <div className="flex items-center gap-1.5 text-xs font-semibold">
            <span className="font-mono text-[#94A3B8]">{orderStr}</span>
            <span className="text-[#94A3B8]">·</span>
            <span style={{ color: statusTextColor }}>{statusText}</span>
          </div>

          <h3 className="mt-0.5 text-sm sm:text-base font-bold text-[#0F172A] tracking-tight line-clamp-1">
            {task.title}
          </h3>

          <div className="mt-0.5 flex items-center gap-1.5 text-xs font-mono">
            <span className="text-[#64748B]">{task.deadline}</span>
            <span className="text-[#94A3B8]">·</span>
            <span style={{ color: subtitleColor }}>{task.subtitle}</span>
          </div>
        </div>

        <div className="shrink-0">
          {isCompleted ? (
            <button
              type="button"
              onClick={() => onToggle(task.id)}
              aria-label={`Задание ${task.order} выполнено. Нажмите для изменения`}
              className="flex h-10 w-10 items-center justify-center rounded-xl border border-[#6EE7B7] bg-[#ECFDF5] text-[#059669] shadow-2xs hover:bg-emerald-100/60 transition-colors cursor-pointer"
            >
              <Check className="h-5 w-5 stroke-[2.8]" />
            </button>
          ) : isError ? (
            <button
              type="button"
              onClick={() => onRetry(task.id)}
              title="Сбой связи! Нажмите, чтобы устранить помехи и повторить"
              aria-label={`Задание ${task.order} имеет сбой. Нажмите для повтора`}
              className="group relative flex h-10 w-10 items-center justify-center rounded-xl border border-[#FDA4AF] bg-[#FFF1F2] text-[#E11D48] shadow-2xs hover:bg-rose-100 transition-colors cursor-pointer"
            >
              <AlertCircle className="h-5 w-5 stroke-[2.5]" />
              <span className="absolute -top-1 -right-1 h-2.5 w-2.5 rounded-full bg-rose-500 animate-ping" />
            </button>
          ) : (
            <button
              type="button"
              onClick={() => onToggle(task.id)}
              aria-label={`Задание ${task.order}. Нажмите, чтобы отметить выполненным`}
              className="flex h-10 w-10 items-center justify-center rounded-xl border border-[#E2E8F0] bg-[#F8FAFC] text-[#CBD5E1] hover:border-[#7DD3FC] hover:text-[#0284C7] hover:bg-sky-50/50 transition-colors cursor-pointer"
            >
              <Check className="h-5 w-5 stroke-[2]" />
            </button>
          )}
        </div>
      </div>
    </div>
  );
});

export default RouteChamferCard;
