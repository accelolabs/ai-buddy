import type { MessageStage } from '@/entities/chat';

export interface ExecutionStagesProps {
  stages: MessageStage[];
  showSkeleton?: boolean;
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

export function ExecutionStages({ stages, showSkeleton = true }: ExecutionStagesProps) {
  if (!stages || stages.length === 0) return null;

  return (
    <div className="flex w-full flex-col gap-3">
      <div className="flex flex-col gap-2.5">
        {stages.map((stage) => {
          const isCompleted = stage.status === 'completed';
          const isInProgress = stage.status === 'in_progress';
          const isError = stage.status === 'error';

          return (
            <div key={stage.id} className="flex items-center gap-3">
              <div
                className={`flex h-6 w-6 shrink-0 items-center justify-center rounded-full transition-all duration-300 ${
                  isCompleted
                    ? 'bg-[#04E061]'
                    : isInProgress
                      ? 'bg-[#00AAFF]'
                      : isError
                        ? 'bg-[#FF4053]'
                        : 'border border-slate-300 bg-slate-100'
                }`}
              >
                {isCompleted && CHECK_ICON}
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
