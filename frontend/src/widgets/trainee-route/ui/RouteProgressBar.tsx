import { memo } from 'react';
import {
  useRouteStore,
  selectCompletedCount,
  selectTotalCount,
  selectProgressPercentage,
  selectFilledSegments,
} from '@/entities/trainee-route';

const SEGMENT_COUNT = 24;

export const RouteProgressBar = memo(function RouteProgressBar() {
  const completedCount = useRouteStore(selectCompletedCount);
  const totalCount = useRouteStore(selectTotalCount);
  const percentage = useRouteStore(selectProgressPercentage);
  const filledSegments = useRouteStore(selectFilledSegments);

  return (
    <div className="w-full">
      {/* 1. Header row */}
      <div className="mb-3 flex items-center justify-between text-sm">
        <span className="font-medium text-[#475569]">Прогресс адаптации</span>
        <div className="font-bold text-[#0F172A]">
          <span>
            {completedCount} из {totalCount}{' '}
          </span>
          <span className="text-[#0284C7]">({percentage}%)</span>
        </div>
      </div>

      {/* 2. Progress Track & Right Plug Connector */}
      <div className="relative flex items-center">
        {/* Track with 24 Slanted Segments */}
        <div className="relative h-7 w-full overflow-hidden rounded-full border border-[#E2E8F0] bg-[#F1F5F9] p-1 shadow-2xs">
          <svg
            viewBox="0 0 771 20"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
            className="h-full w-full select-none"
            preserveAspectRatio="none"
          >
            {Array.from({ length: SEGMENT_COUNT }).map((_, index) => {
              const isFilled = index < filledSegments;
              const startX = 6 + index * 31.8;
              const width = 23;
              // Slanted parallelogram (10-degree tilt matching route.svg)
              const d = `M${startX + 3} 0 H${startX + 3 + width} L${startX + width} 20 H${startX} Z`;

              return (
                <path
                  key={index}
                  d={d}
                  fill={isFilled ? '#0284C7' : '#E2E8F0'}
                  stroke={isFilled ? 'none' : '#CBD5E1'}
                  strokeWidth="0.8"
                  className="transition-colors duration-300"
                />
              );
            })}
          </svg>
        </div>

        {/* Right Chip Plug Connector (route.svg elements 18-21) */}
        <div className="hidden sm:flex shrink-0 items-center pl-2">
          <svg
            width="42"
            height="28"
            viewBox="0 0 42 28"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
            className="overflow-visible"
          >
            {/* Plug bracket on progress bar */}
            <path
              d="M0 6H6C8.2 6 10 7.8 10 10V18C10 20.2 8.2 22 6 22H0"
              fill="#CBD5E1"
            />
            {/* Socket terminal */}
            <rect
              x="11"
              y="9"
              width="8"
              height="10"
              rx="2"
              fill="#F8FAFC"
              stroke="#94A3B8"
              strokeWidth="1.2"
            />
            {/* Outgoing circuit trace going down and right */}
            <path
              d="M19 14H28C31 14 33 16 33 19V25"
              stroke="#CBD5E1"
              strokeWidth="1.5"
            />
            {/* Terminal contact pad */}
            <rect
              x="29"
              y="23"
              width="8"
              height="4"
              rx="1"
              fill="#F0F9FF"
              stroke="#7DD3FC"
              strokeWidth="1"
            />
          </svg>
        </div>
      </div>
    </div>
  );
});

export default RouteProgressBar;
