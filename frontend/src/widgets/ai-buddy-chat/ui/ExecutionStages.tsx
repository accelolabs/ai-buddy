import { useState, memo } from 'react';
import { Check, Loader2, ChevronDown } from 'lucide-react';
import type { MessageStage } from '@/entities/chat';

export interface ExecutionStagesProps {
  stages: MessageStage[];
  showSkeleton?: boolean;
  isCompleted?: boolean;
}

export const ExecutionStages = memo(function ExecutionStages({
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
            <Check className="h-3 w-3 text-[#04E061]" />
          </span>
          <span>
            {stages.length === 1
              ? 'Выполнен 1 этап'
              : `Выполнено этапов: ${stages.length}`}
          </span>
          <ChevronDown
            className={`h-3.5 w-3.5 transition-transform duration-200 ${isOpen ? 'rotate-180' : ''}`}
          />
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
                  {isStageDone && <Check className="h-3.5 w-3.5 text-white stroke-[2.5]" />}
                  {isInProgress && <Loader2 className="h-3.5 w-3.5 animate-spin text-white" />}
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
});

export default ExecutionStages;
