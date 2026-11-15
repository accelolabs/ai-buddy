import { memo } from 'react';
import { useRouteStore } from '@/entities/trainee-route';

export const RouteInternTabs = memo(function RouteInternTabs() {
  const interns = useRouteStore((s) => s.interns);
  const activeInternId = useRouteStore((s) => s.activeInternId);
  const setActiveIntern = useRouteStore((s) => s.setActiveIntern);

  return (
    <div className="mb-6 w-full">
      {/* 1. Page Title */}
      <h1 className="mb-6 text-[32px] font-bold tracking-tight text-[#0F172A] leading-tight">
        Мои стажеры
      </h1>

      {/* 2. Tabs Row */}
      <div className="flex flex-wrap items-center gap-3">
        {interns.map((intern) => {
          const isActive = intern.id === activeInternId;

          if (isActive) {
            return (
              <button
                key={intern.id}
                type="button"
                onClick={() => setActiveIntern(intern.id)}
                className="relative inline-flex h-[38px] items-center gap-2 rounded-xl border border-[#7DD3FC] bg-[#F0F9FF] px-4 font-semibold text-xs sm:text-sm text-[#0C4A6E] shadow-2xs transition-all cursor-pointer"
              >
                {/* Chip / Plug Pin Icon from route.svg */}
                <svg
                  width="14"
                  height="14"
                  viewBox="0 0 16 16"
                  fill="none"
                  xmlns="http://www.w3.org/2000/svg"
                  className="shrink-0 text-[#0EA5E9]"
                >
                  <rect
                    x="2.5"
                    y="4.5"
                    width="11"
                    height="8"
                    rx="1.5"
                    stroke="#0EA5E9"
                    strokeWidth="1.5"
                  />
                  <line
                    x1="5.5"
                    y1="1.5"
                    x2="5.5"
                    y2="4.5"
                    stroke="#0EA5E9"
                    strokeWidth="1.5"
                    strokeLinecap="round"
                  />
                  <line
                    x1="10.5"
                    y1="1.5"
                    x2="10.5"
                    y2="4.5"
                    stroke="#0EA5E9"
                    strokeWidth="1.5"
                    strokeLinecap="round"
                  />
                  <circle cx="8" cy="8.5" r="1.5" fill="#0EA5E9" />
                </svg>
                <span>{intern.name}</span>
              </button>
            );
          }

          return (
            <button
              key={intern.id}
              type="button"
              onClick={() => setActiveIntern(intern.id)}
              className="relative inline-flex h-[38px] items-center rounded-xl border border-[#E2E8F0] bg-white px-4 font-medium text-xs sm:text-sm text-[#334155] hover:bg-slate-50 transition-colors cursor-pointer"
            >
              <span>{intern.name}</span>
            </button>
          );
        })}
      </div>
    </div>
  );
});

export default RouteInternTabs;
