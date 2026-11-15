import { memo } from 'react';
import { useRouteStore, selectActiveIntern } from '@/entities/trainee-route';

export const RouteGoalCards = memo(function RouteGoalCards() {
  const intern = useRouteStore(selectActiveIntern);

  return (
    <div className="relative w-full">
      {/* 1. Two Summary Cards */}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 md:gap-6">
        {/* Card Left: ЦЕЛЬ */}
        <div className="relative flex flex-col justify-between rounded-2xl border border-[#E2E8F0] bg-[#F1F5F9] p-5 shadow-2xs min-h-[109px]">
          {/* 3 Left Terminal Pins (route.svg elements 25-27) */}
          <div className="absolute -left-1.5 top-1/2 -translate-y-1/2 flex flex-col gap-2.5">
            <span className="h-1 w-2 rounded-r-xs bg-[#CBD5E1]" />
            <span className="h-1 w-2 rounded-r-xs bg-[#CBD5E1]" />
            <span className="h-1 w-2 rounded-r-xs bg-[#CBD5E1]" />
          </div>

          <div>
            <span className="font-mono text-xs font-semibold tracking-wider uppercase text-[#94A3B8]">
              ЦЕЛЬ
            </span>
            <p className="mt-1 text-sm font-medium leading-snug text-[#1E293B]">
              {intern.goal}
            </p>
          </div>

          {/* Bottom Chip Socket Pad */}
          <div className="absolute -bottom-1.5 left-1/2 -translate-x-1/2 flex items-center justify-center">
            <div className="h-1.5 w-5 rounded-xs border border-[#CBD5E1] bg-[#E2E8F0]" />
          </div>
        </div>

        {/* Card Right: ОЖИДАНИЯ */}
        <div className="relative flex flex-col justify-between rounded-2xl border border-[#E2E8F0] bg-[#F1F5F9] p-5 shadow-2xs min-h-[109px]">
          {/* 3 Left Terminal Pins */}
          <div className="absolute -left-1.5 top-1/2 -translate-y-1/2 flex flex-col gap-2.5">
            <span className="h-1 w-2 rounded-r-xs bg-[#CBD5E1]" />
            <span className="h-1 w-2 rounded-r-xs bg-[#CBD5E1]" />
            <span className="h-1 w-2 rounded-r-xs bg-[#CBD5E1]" />
          </div>

          <div>
            <span className="font-mono text-xs font-semibold tracking-wider uppercase text-[#94A3B8]">
              ОЖИДАНИЯ
            </span>
            <p className="mt-1 text-sm font-medium leading-snug text-[#1E293B]">
              {intern.expectations}
            </p>
          </div>

          {/* Bottom Chip Socket Pad */}
          <div className="absolute -bottom-1.5 left-1/2 -translate-x-1/2 flex items-center justify-center">
            <div className="h-1.5 w-5 rounded-xs border border-[#CBD5E1] bg-[#E2E8F0]" />
          </div>
        </div>
      </div>

      {/* 2. Bus Connecting Traces between Summary Cards and Main Board (route.svg elements 39-50) */}
      <div className="relative h-12 w-full overflow-visible pointer-events-none select-none">
        <svg
          viewBox="0 0 942 50"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
          className="h-full w-full"
          preserveAspectRatio="none"
        >
          {/* Left Bus Trace: from Left Card (x=245.156) to Left Board Port (x=423.35) */}
          <path
            d="M245.156 0V15C245.156 22 248.3 25 254.6 25H413.9C420.2 25 423.35 28 423.35 35V50"
            stroke="#CBD5E1"
            strokeWidth="1.8"
            strokeLinecap="round"
          />
          {/* Animated Electrical Current on Left Trace */}
          <path
            d="M245.156 0V15C245.156 22 248.3 25 254.6 25H413.9C420.2 25 423.35 28 423.35 35V50"
            stroke="#0284C7"
            strokeWidth="1.8"
            strokeLinecap="round"
            pathLength="1"
            className="animate-electric-flow"
          />

          {/* Right Bus Trace: from Right Card (x=686.843) to Right Board Port (x=508.65) */}
          <path
            d="M686.843 0V15C686.843 22 683.7 25 677.4 25H518.1C511.8 25 508.65 28 508.65 35V50"
            stroke="#CBD5E1"
            strokeWidth="1.8"
            strokeLinecap="round"
          />
          {/* Animated Electrical Current on Right Trace */}
          <path
            d="M686.843 0V15C686.843 22 683.7 25 677.4 25H518.1C511.8 25 508.65 28 508.65 35V50"
            stroke="#0284C7"
            strokeWidth="1.8"
            strokeLinecap="round"
            pathLength="1"
            className="animate-electric-flow-fast"
            style={{ animationDelay: '0.5s' }}
          />

          {/* Top Contact Pads */}
          <circle cx="245.156" cy="5" r="3" fill="#94A3B8" />
          <circle cx="245.156" cy="5" r="1.2" fill="#FFFFFF" />
          <circle cx="686.843" cy="5" r="3" fill="#94A3B8" />
          <circle cx="686.843" cy="5" r="1.2" fill="#FFFFFF" />

          {/* Bottom Terminal Ports entering Main Board */}
          {/* Left Board Port */}
          <rect
            x="414"
            y="42"
            width="19"
            height="7"
            rx="1.5"
            fill="#F0F9FF"
            stroke="#7DD3FC"
            strokeWidth="1"
          />
          <circle cx="423.35" cy="45.5" r="2.5" fill="#0EA5E9" />
          <circle cx="423.35" cy="45.5" r="1" fill="#FFFFFF" />

          {/* Right Board Port */}
          <rect
            x="499"
            y="42"
            width="19"
            height="7"
            rx="1.5"
            fill="#F0F9FF"
            stroke="#7DD3FC"
            strokeWidth="1"
          />
          <circle cx="508.65" cy="45.5" r="2.5" fill="#0EA5E9" />
          <circle cx="508.65" cy="45.5" r="1" fill="#FFFFFF" />
        </svg>
      </div>
    </div>
  );
});

export default RouteGoalCards;
