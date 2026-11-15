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

        {/* Bottom Center Contact Terminal to wire leading to Card 1 */}
        <div className="pointer-events-none absolute -bottom-1.5 left-1/2 -translate-x-1/2 z-10 flex items-center justify-center">
          <div className="h-1.5 w-5 rounded-xs border border-[#CBD5E1] bg-[#F1F5F9]" />
        </div>
      </div>
    </div>
  );
});

export default RouteProgressBar;
