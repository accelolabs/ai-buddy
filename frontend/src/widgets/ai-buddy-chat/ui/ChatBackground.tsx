interface CircuitTrace {
  id: string;
  path: string;
  startY: number;
  dot: { cx: number; cy: number };
  color: string;
  delay: string;
}

const LEFT_CIRCUIT_CONFIG: CircuitTrace[] = [
  {
    id: 'left-blue-1',
    startY: 55,
    path: 'M 0 55 H 250 C 275 55 290 70 290 95 V 140 C 290 160 305 175 325 175 H 410',
    dot: { cx: 410, cy: 175 },
    color: '#0EA5E9',
    delay: '0s',
  },
  {
    id: 'left-green-1',
    startY: 135,
    path: 'M 0 135 H 150 C 175 135 190 150 190 175 V 230 C 190 250 205 265 225 265 H 290',
    dot: { cx: 290, cy: 265 },
    color: '#10B981',
    delay: '1.4s',
  },
  {
    id: 'left-purple-1',
    startY: 240,
    path: 'M 0 240 H 260 C 285 240 300 225 300 200 V 170',
    dot: { cx: 300, cy: 170 },
    color: '#8B5CF6',
    delay: '2.6s',
  },
  {
    id: 'left-blue-2',
    startY: 350,
    path: 'M 0 350 H 210 C 235 350 250 365 250 390 V 460 C 250 480 265 495 285 495 H 380',
    dot: { cx: 380, cy: 495 },
    color: '#0EA5E9',
    delay: '0.8s',
  },
  {
    id: 'left-green-2',
    startY: 470,
    path: 'M 0 470 H 130 C 155 470 170 485 170 510 V 580 C 170 600 185 615 205 615 H 240',
    dot: { cx: 240, cy: 615 },
    color: '#10B981',
    delay: '2.1s',
  },
  {
    id: 'left-red-1',
    startY: 575,
    path: 'M 0 575 H 240 C 265 575 280 590 280 615 V 680 C 280 700 295 715 315 715 H 395',
    dot: { cx: 395, cy: 715 },
    color: '#F43F5E',
    delay: '3.2s',
  },
  {
    id: 'left-blue-3',
    startY: 690,
    path: 'M 0 690 H 170 C 195 690 210 705 210 730 V 790',
    dot: { cx: 210, cy: 790 },
    color: '#0EA5E9',
    delay: '1.7s',
  },
  {
    id: 'left-purple-2',
    startY: 780,
    path: 'M 0 780 H 110 C 130 780 145 765 145 745 V 720',
    dot: { cx: 145, cy: 720 },
    color: '#8B5CF6',
    delay: '2.9s',
  },
];

const RIGHT_CIRCUIT_CONFIG: CircuitTrace[] = [
  {
    id: 'right-purple-1',
    startY: 85,
    path: 'M 340 85 H 180 C 155 85 140 100 140 125 V 195',
    dot: { cx: 140, cy: 195 },
    color: '#8B5CF6',
    delay: '0.9s',
  },
  {
    id: 'right-red-1',
    startY: 270,
    path: 'M 340 270 H 190 C 165 270 150 285 150 310 V 380 C 150 400 135 415 115 415 H 60',
    dot: { cx: 60, cy: 415 },
    color: '#F43F5E',
    delay: '2.2s',
  },
  {
    id: 'right-green-1',
    startY: 540,
    path: 'M 340 540 H 170 C 145 540 130 555 130 580 V 670',
    dot: { cx: 130, cy: 670 },
    color: '#10B981',
    delay: '0.4s',
  },
  {
    id: 'right-blue-1',
    startY: 680,
    path: 'M 340 680 H 220 C 195 680 180 665 180 640 V 605',
    dot: { cx: 180, cy: 605 },
    color: '#0EA5E9',
    delay: '1.8s',
  },
];

export function ChatBackground() {
  return (
    <div aria-hidden="true" className="pointer-events-none absolute inset-0 z-0 overflow-hidden">
      <style>{`
        @keyframes neon-pulse {
          0%, 100% { opacity: 0.35; filter: drop-shadow(0 0 2px currentColor); }
          50% { opacity: 1; filter: drop-shadow(0 0 6px currentColor); }
        }
        @keyframes dash-flow {
          0% { stroke-dashoffset: 1; }
          100% { stroke-dashoffset: 0; }
        }
        .animate-neon {
          animation: neon-pulse 3s infinite ease-in-out;
        }
        .animate-flow {
          stroke-dasharray: 0.3 0.7;
          animation: dash-flow 4.5s infinite linear;
        }
      `}</style>

      {/* Left Circuit: Originates from sidebar, filling the space left of chat */}
      <div
        className="absolute left-0 top-0 bottom-0 hidden overflow-hidden min-[900px]:block"
        style={{ width: 'calc(50% - 370px)', maxWidth: '480px' }}
      >
        <svg
          viewBox="0 0 460 860"
          preserveAspectRatio="xMinYMid meet"
          className="h-full w-full"
          role="presentation"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
        >
          <defs>
            <filter id="neonGlowLeft" x="-30%" y="-30%" width="160%" height="160%">
              <feGaussianBlur stdDeviation="3.5" result="blur" />
              <feMerge>
                <feMergeNode in="blur" />
                <feMergeNode in="SourceGraphic" />
              </feMerge>
            </filter>
          </defs>

          {/* Base circuit traces */}
          <g
            stroke="#94A3B8"
            strokeWidth="1.6"
            strokeLinecap="round"
            strokeLinejoin="round"
            opacity="0.25"
          >
            {LEFT_CIRCUIT_CONFIG.map((item) => (
              <path key={`base-${item.id}`} d={item.path} />
            ))}
          </g>

          {/* Sidebar contact pins */}
          <g fill="#94A3B8" opacity="0.45">
            {LEFT_CIRCUIT_CONFIG.map((item) => (
              <rect
                key={`pin-${item.id}`}
                x="0"
                y={item.startY - 3}
                width="6"
                height="6"
                rx="1.5"
              />
            ))}
          </g>

          {/* Animated glowing pulses and terminal nodes */}
          {LEFT_CIRCUIT_CONFIG.map((item) => (
            <g
              key={`animated-${item.id}`}
              style={{
                color: item.color,
                animationDelay: item.delay,
              }}
              className="animate-neon"
            >
              <path
                d={item.path}
                stroke="currentColor"
                strokeWidth="2.2"
                strokeLinecap="round"
                strokeLinejoin="round"
                pathLength="1"
                className="animate-flow"
                style={{ animationDelay: item.delay }}
              />

              {/* Glowing halo */}
              <circle
                cx={item.dot.cx}
                cy={item.dot.cy}
                r="8"
                fill="currentColor"
                opacity="0.35"
                filter="url(#neonGlowLeft)"
              />
              {/* Outer dot */}
              <circle cx={item.dot.cx} cy={item.dot.cy} r="4" fill="currentColor" />
              {/* Bright center core */}
              <circle cx={item.dot.cx} cy={item.dot.cy} r="1.5" fill="#FFFFFF" opacity="0.9" />
            </g>
          ))}
        </svg>
      </div>

      {/* Right Circuit: Symmetrical subtle lines from right edge, leaving chat area clear */}
      <div
        className="absolute right-0 top-0 bottom-0 hidden overflow-hidden min-[900px]:block"
        style={{ width: 'calc(50% - 370px)', maxWidth: '360px' }}
      >
        <svg
          viewBox="0 0 340 860"
          preserveAspectRatio="xMaxYMid meet"
          className="h-full w-full"
          role="presentation"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
        >
          <defs>
            <filter id="neonGlowRight" x="-30%" y="-30%" width="160%" height="160%">
              <feGaussianBlur stdDeviation="3.5" result="blur" />
              <feMerge>
                <feMergeNode in="blur" />
                <feMergeNode in="SourceGraphic" />
              </feMerge>
            </filter>
          </defs>

          {/* Base circuit traces */}
          <g
            stroke="#94A3B8"
            strokeWidth="1.6"
            strokeLinecap="round"
            strokeLinejoin="round"
            opacity="0.25"
          >
            {RIGHT_CIRCUIT_CONFIG.map((item) => (
              <path key={`base-${item.id}`} d={item.path} />
            ))}
          </g>

          {/* Right edge contact pins */}
          <g fill="#94A3B8" opacity="0.45">
            {RIGHT_CIRCUIT_CONFIG.map((item) => (
              <rect
                key={`pin-${item.id}`}
                x="334"
                y={item.startY - 3}
                width="6"
                height="6"
                rx="1.5"
              />
            ))}
          </g>

          {/* Animated glowing pulses and terminal nodes */}
          {RIGHT_CIRCUIT_CONFIG.map((item) => (
            <g
              key={`animated-${item.id}`}
              style={{
                color: item.color,
                animationDelay: item.delay,
              }}
              className="animate-neon"
            >
              <path
                d={item.path}
                stroke="currentColor"
                strokeWidth="2.2"
                strokeLinecap="round"
                strokeLinejoin="round"
                pathLength="1"
                className="animate-flow"
                style={{ animationDelay: item.delay }}
              />

              {/* Glowing halo */}
              <circle
                cx={item.dot.cx}
                cy={item.dot.cy}
                r="8"
                fill="currentColor"
                opacity="0.35"
                filter="url(#neonGlowRight)"
              />
              {/* Outer dot */}
              <circle cx={item.dot.cx} cy={item.dot.cy} r="4" fill="currentColor" />
              {/* Bright center core */}
              <circle cx={item.dot.cx} cy={item.dot.cy} r="1.5" fill="#FFFFFF" opacity="0.9" />
            </g>
          ))}
        </svg>
      </div>
    </div>
  );
}
