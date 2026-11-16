import { memo } from 'react';
import { LEFT_CIRCUIT_CONFIG, RIGHT_CIRCUIT_CONFIG } from '../constants/backgroundCircuits';
import { CHAT_CIRCUIT_ANIMATION_STYLES } from '../constants/chatStyles';

export const ChatBackground = memo(function ChatBackground() {
  return (
    <div aria-hidden="true" className="pointer-events-none absolute inset-0 z-0 overflow-hidden">
      <style>{CHAT_CIRCUIT_ANIMATION_STYLES}</style>

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

              <circle
                cx={item.dot.cx}
                cy={item.dot.cy}
                r="8"
                fill="currentColor"
                opacity="0.35"
                filter="url(#neonGlowLeft)"
              />
              <circle cx={item.dot.cx} cy={item.dot.cy} r="4" fill="currentColor" />
              <circle cx={item.dot.cx} cy={item.dot.cy} r="1.5" fill="#FFFFFF" opacity="0.9" />
            </g>
          ))}
        </svg>
      </div>

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

              <circle
                cx={item.dot.cx}
                cy={item.dot.cy}
                r="8"
                fill="currentColor"
                opacity="0.35"
                filter="url(#neonGlowRight)"
              />
              <circle cx={item.dot.cx} cy={item.dot.cy} r="4" fill="currentColor" />
              <circle cx={item.dot.cx} cy={item.dot.cy} r="1.5" fill="#FFFFFF" opacity="0.9" />
            </g>
          ))}
        </svg>
      </div>
    </div>
  );
});

export default ChatBackground;
