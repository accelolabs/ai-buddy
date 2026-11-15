import { memo } from 'react';
import { useRouteStore, selectActiveIntern } from '@/entities/trainee-route';
import { RouteProgressBar } from './RouteProgressBar';
import { RouteChamferCard } from './RouteChamferCard';
import { RouteWireSegment, type WireVariant } from './RouteWireSegment';

export const MainRouteBoard = memo(function MainRouteBoard() {
  const intern = useRouteStore(selectActiveIntern);
  const toggleTask = useRouteStore((s) => s.toggleTask);
  const retryTask = useRouteStore((s) => s.retryTask);

  const tasks = intern.tasks;

  return (
    <div className="relative w-full overflow-hidden rounded-3xl border border-[#E2E8F0] bg-white p-6 sm:p-8 shadow-xs">
      <svg
        width="16"
        height="16"
        viewBox="0 0 16 16"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className="pointer-events-none absolute left-4 top-4 select-none"
      >
        <path
          d="M1 15V4C1 2.34315 2.34315 1 4 1H15"
          stroke="#7DD3FC"
          strokeWidth="2"
          strokeLinecap="round"
        />
      </svg>

      <svg
        width="16"
        height="16"
        viewBox="0 0 16 16"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className="pointer-events-none absolute right-4 top-4 select-none"
      >
        <path
          d="M15 15V4C15 2.34315 13.6569 1 12 1H1"
          stroke="#7DD3FC"
          strokeWidth="2"
          strokeLinecap="round"
        />
      </svg>

      <div className="mb-6 flex flex-col">
        <span className="text-xs font-bold tracking-wider uppercase text-[#0284C7]">
          {intern.role}
        </span>

        <h2 className="mt-1 text-2xl sm:text-[28px] font-bold tracking-tight text-[#0F172A]">
          {intern.name}
        </h2>

        <p className="mt-1 text-xs text-[#64748B]">
          <span>Наставник: </span>
          <span className="font-semibold text-[#1E293B]">{intern.mentor}</span>
          <span className="mx-2 text-[#CBD5E1]">·</span>
          <span>Дата старта: </span>
          <span className="font-semibold text-[#1E293B]">{intern.startDate}</span>
        </p>
      </div>

      <div className="w-full">
        <RouteProgressBar />
      </div>

      <RouteWireSegment variant={tasks[0]?.completed ? 'green' : 'gray'} />

      <div className="flex flex-col items-center w-full">
        {tasks.map((task, index) => {
          const hasNext = index < tasks.length - 1;

          let wireVariant: WireVariant = 'gray';
          if (task.completed) {
            wireVariant = 'green';
          } else if (task.status === 'error' || task.hasError) {
            wireVariant = 'glitch';
          }

          return (
            <div key={task.id} className="w-full flex flex-col items-center">
              <RouteChamferCard task={task} onToggle={toggleTask} onRetry={retryTask} />

              {hasNext && <RouteWireSegment variant={wireVariant} isGlitching={task.isGlitching} />}
            </div>
          );
        })}
      </div>
    </div>
  );
});

export default MainRouteBoard;
