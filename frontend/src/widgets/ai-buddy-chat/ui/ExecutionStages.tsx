import { useState } from 'react';
import type { MessageStage } from '@/entities/chat';

export interface ExecutionStagesProps {
  stages: MessageStage[];
  showSkeleton?: boolean;
  isCompleted?: boolean;
}

const CHECK_ICON = (
  <svg width="14" height="14" viewBox="0 0 14 14" fill="none" className="text-black">
    <path
      d="M10.5 4.25L4.542 10.208L1.833 7.5"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
    <path
      d="M12.667 6.417L8.604 10.479L7.792 9.667"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
  </svg>
);

const SPINNER_ICON = (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" className="animate-spin text-black">
    <path
      d="M12 2V6M12 18V22M4.93 4.93L7.76 7.76M16.24 16.24L19.07 19.07M2 12H6M18 12H22M4.93 19.07L7.76 16.24M16.24 7.76L19.07 4.93"
      stroke="currentColor"
      strokeWidth="2.5"
      strokeLinecap="round"
      strokeLinejoin="round"
    />
  </svg>
);

const CHEVRON_DOWN_ICON = (
  <svg
    className="h-3.5 w-3.5 transition-transform duration-200"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <polyline points="6 9 12 15 18 9" />
  </svg>
);

export function ExecutionStages({
  stages,
  showSkeleton = true,
  isCompleted = false,
}: ExecutionStagesProps) {
  const [isOpen, setIsOpen] = useState(true);

  if (!stages || stages.length === 0) return null;

  const allDone = isCompleted || stages.every((s) => s.status === 'completed');

  return (
    <div className="flex w-full flex-col gap-2">
      {allDone && (
        <button
          type="button"
          onClick={() => setIsOpen((prev) => !prev)}
          className="inline-flex w-fit items-center gap-1.5 rounded-lg py-0.5 text-xs font-semibold text-[#64748B] transition-colors hover:text-[#0F172A]"
        >
          <span className="flex h-4 w-4 items-center justify-center rounded-full bg-emerald-100 text-[#04E061]">
            ✓
          </span>
          <span>
            {stages.length === 1
              ? 'Выполнен 1 этап'
              : `Выполнено этапов: ${stages.length}`}
          </span>
          <span className={isOpen ? 'rotate-180' : ''}>{CHEVRON_DOWN_ICON}</span>
        </button>
      )}

      {isOpen && (
        <div className="flex flex-col gap-2.5 pt-0.5">
          {stages.map((stage) => {
            const isStageDone = stage.status === 'completed';
            const isInProgress = stage.status === 'in_progress';
            const isError = stage.status === 'error';

            return (
              <div key={stage.id} className="flex items-center gap-3">
                <div
                  className={`flex h-6 w-6 shrink-0 items-center justify-center rounded-full transition-all duration-300 ${
                    isStageDone
                      ? 'bg-[#04E061]'
                      : isInProgress
                        ? 'bg-[#00AAFF]'
                        : isError
                          ? 'bg-[#FF4053]'
                          : 'border border-slate-300 bg-slate-100'
                  }`}
                >
                  {isStageDone && CHECK_ICON}
                  {isInProgress && SPINNER_ICON}
                  {isError && <span className="text-xs font-bold text-white">!</span>}
                </div>
                <span
                  className={`text-[15px] ${
                    isInProgress ? 'font-semibold text-[#0F172A]' : 'font-medium text-[#0F172A]'
                  }`}
                >
                  {stage.label}
                </span>
              </div>
            );
          })}
        </div>
      )}

      {showSkeleton && (
        <div className="mt-2 flex flex-col gap-2.5 pt-1">
          <div className="h-2.5 w-full rounded-full bg-[#E5E9F2] animate-pulse" />
          <div className="h-2.5 w-[88%] rounded-full bg-[#E5E9F2] animate-pulse" />
          <div className="h-2.5 w-[65%] rounded-full bg-[#E5E9F2] animate-pulse" />
        </div>
      )}
    </div>
  );
}

export default ExecutionStages;
