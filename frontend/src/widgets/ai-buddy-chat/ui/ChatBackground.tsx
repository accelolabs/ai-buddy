import { memo, useEffect, useRef } from 'react';
import { LEFT_CIRCUIT_CONFIG, RIGHT_CIRCUIT_CONFIG } from '../constants/backgroundCircuits';
import './ChatBackground.css';

const SIDES = [
  { id: 'left', traces: LEFT_CIRCUIT_CONFIG, width: 460, maxWidth: 480, pinX: 0 },
  { id: 'right', traces: RIGHT_CIRCUIT_CONFIG, width: 340, maxWidth: 360, pinX: 334 },
].map((side) => ({ ...side, basePath: side.traces.map((trace) => trace.path).join(' ') }));

export const ChatBackground = memo(function ChatBackground() {
  const background = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const syncVisibility = () =>
      background.current?.toggleAttribute('data-paused', document.hidden);
    syncVisibility();
    document.addEventListener('visibilitychange', syncVisibility);
    return () => document.removeEventListener('visibilitychange', syncVisibility);
  }, []);

  return (
    <div
      ref={background}
      aria-hidden="true"
      className="chat-background pointer-events-none fixed inset-x-0 bottom-0 top-20 z-0 hidden overflow-hidden min-[900px]:block"
    >
      {SIDES.map((side) => (
        <div
          key={side.id}
          className="absolute inset-y-0 overflow-hidden"
          style={{ [side.id]: 0, width: 'calc(50% - 370px)', maxWidth: side.maxWidth }}
        >
          <svg
            viewBox={`0 0 ${side.width} 860`}
            preserveAspectRatio={side.id === 'left' ? 'xMinYMid meet' : 'xMaxYMid meet'}
            className="h-full w-full"
            role="presentation"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
          >
            <defs>
              <filter id={`chat-neon-glow-${side.id}`} x="-30%" y="-30%" width="160%" height="160%">
                <feGaussianBlur stdDeviation="3.5" result="blur" />
                <feMerge>
                  <feMergeNode in="blur" />
                  <feMergeNode in="SourceGraphic" />
                </feMerge>
              </filter>
            </defs>
            <g
              stroke="#94A3B8"
              strokeWidth="1.6"
              strokeLinecap="round"
              strokeLinejoin="round"
              opacity="0.25"
            >
              <path d={side.basePath} />
            </g>
            <g fill="#94A3B8" opacity="0.45">
              {side.traces.map((trace) => (
                <rect
                  key={trace.id}
                  x={side.pinX}
                  y={trace.startY - 3}
                  width="6"
                  height="6"
                  rx="1.5"
                />
              ))}
            </g>
            {side.traces.map((trace) => (
              <g
                key={trace.id}
                style={{ color: trace.color, animationDelay: trace.delay }}
                className="chat-circuit-pulse"
              >
                <path
                  d={trace.path}
                  stroke="currentColor"
                  strokeWidth="2.2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  pathLength="1"
                  className="chat-circuit-flow"
                  style={{ animationDelay: trace.delay }}
                />
                <circle
                  cx={trace.dot.cx}
                  cy={trace.dot.cy}
                  r="8"
                  fill="currentColor"
                  opacity="0.35"
                  filter={`url(#chat-neon-glow-${side.id})`}
                />
                <circle cx={trace.dot.cx} cy={trace.dot.cy} r="4" fill="currentColor" />
                <circle cx={trace.dot.cx} cy={trace.dot.cy} r="1.5" fill="#FFFFFF" opacity="0.9" />
              </g>
            ))}
          </svg>
        </div>
      ))}
    </div>
  );
});
