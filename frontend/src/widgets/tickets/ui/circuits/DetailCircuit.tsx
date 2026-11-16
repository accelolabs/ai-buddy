import { memo } from 'react';

export const DetailCircuit = memo(function DetailCircuit() {
  return (
    <div className="pointer-events-none absolute inset-0 z-0 overflow-hidden rounded-[16px]">
      <svg
        viewBox="24 104 595 585"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className="h-full w-full"
        preserveAspectRatio="none"
      >
        <defs>
          <filter id="detailNeonGlow" x="-30%" y="-30%" width="160%" height="160%">
            <feGaussianBlur stdDeviation="3.5" result="blur" />
            <feMerge>
              <feMergeNode in="blur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>
        </defs>

        <g stroke="#94A3B8" strokeWidth="2.45" strokeLinecap="round" opacity="0.3">
          <path d="M640.302 181.184H435.126C396.47 181.184 377.142 200.604 377.142 239.443V306.665" />
          <path d="M640.302 593.475H479.73C441.073 593.475 421.745 612.895 421.745 651.734V709.993" />
          <path d="M20.3147 709.993V642.771C20.3147 615.882 33.6957 602.438 60.4577 602.438H180.887" />
          <path
            d="M640.302 181.184H638.319H586.778"
            strokeWidth="3.35"
            stroke="#64748B"
            opacity="0.45"
          />
        </g>

        <g style={{ color: '#0EA5E9' }} className="animate-neon">
          <path
            d="M640.302 181.184H435.126C396.47 181.184 377.142 200.604 377.142 239.443V306.665"
            stroke="currentColor"
            strokeWidth="2.45"
            strokeLinecap="round"
            pathLength="1"
            className="animate-flow"
            style={{ animationDuration: '3.8s' }}
          />
          <path
            d="M640.302 593.475H479.73C441.073 593.475 421.745 612.895 421.745 651.734V709.993"
            stroke="currentColor"
            strokeWidth="2.45"
            strokeLinecap="round"
            pathLength="1"
            className="animate-flow"
            style={{ animationDuration: '4.4s', animationDelay: '1.2s' }}
          />
          <path
            d="M20.3147 709.993V642.771C20.3147 615.882 33.6957 602.438 60.4577 602.438H180.887"
            stroke="currentColor"
            strokeWidth="2.45"
            strokeLinecap="round"
            pathLength="1"
            className="animate-flow"
            style={{ animationDuration: '4s', animationDelay: '0.6s' }}
          />

          <circle
            cx="377.142"
            cy="306.664"
            r="8"
            fill="currentColor"
            opacity="0.35"
            filter="url(#detailNeonGlow)"
          />
          <circle cx="377.142" cy="306.664" r="4.5" fill="currentColor" />
          <circle cx="377.142" cy="306.664" r="1.8" fill="#FFFFFF" opacity="0.9" />

          <circle
            cx="479.73"
            cy="593.475"
            r="8"
            fill="currentColor"
            opacity="0.35"
            filter="url(#detailNeonGlow)"
          />
          <circle cx="479.73" cy="593.475" r="4.5" fill="currentColor" />
          <circle cx="479.73" cy="593.475" r="1.8" fill="#FFFFFF" opacity="0.9" />

          <circle
            cx="180.887"
            cy="602.438"
            r="8"
            fill="currentColor"
            opacity="0.35"
            filter="url(#detailNeonGlow)"
          />
          <circle cx="180.887" cy="602.438" r="4.5" fill="currentColor" />
          <circle cx="180.887" cy="602.438" r="1.8" fill="#FFFFFF" opacity="0.9" />
        </g>
      </svg>
    </div>
  );
});
