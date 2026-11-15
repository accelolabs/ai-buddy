import { memo } from 'react';

export type WireVariant = 'green' | 'glitch' | 'gray';

export interface RouteWireSegmentProps {
  variant: WireVariant;
  isGlitching?: boolean;
}

export const RouteWireSegment = memo(function RouteWireSegment({
  variant,
  isGlitching = false,
}: RouteWireSegmentProps) {
  if (variant === 'green') {
    return (
      <div className="flex h-8 w-full items-center justify-center select-none pointer-events-none">
        <svg
          width="2"
          height="32"
          viewBox="0 0 2 32"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
        >
          <rect width="2" height="32" fill="#34D399" />
        </svg>
      </div>
    );
  }

  if (variant === 'glitch') {
    return (
      <div
        className={`flex h-9 w-full items-center justify-center select-none pointer-events-none overflow-visible ${
          isGlitching ? 'animate-glitch-shake' : ''
        }`}
      >
        <svg
          width="24"
          height="36"
          viewBox="456 908 24 36"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
          className="overflow-visible"
        >
          <path
            d="M464 908.05V910.05M464 944.05V942.3M464 914.05V916.05L462.5 917.3M459.5 919.8L458 921.05L459.375 921.8M462.125 923.3L464.875 924.8M467.625 926.3L469 927.05L466.5 928.55M461.5 931.55L459 933.05L460.25 934.05M462.75 936.05L464 937.05V938.8"
            strokeWidth="1.6"
            stroke="#00D8FF"
            strokeOpacity="0.85"
            strokeLinecap="round"
            className="animate-pulse"
          />
          <path
            d="M467 908.05V910.55M467 944.05V941.862M467 913.55V916.05L465.125 917.612M462.875 919.487L461 921.05L462.719 921.987M464.781 923.112L468.219 924.987M470.281 926.112L472 927.05L468.875 928.925M465.125 931.175L462 933.05L463.562 934.3M465.438 935.8L467 937.05V939.237"
            strokeWidth="1.8"
            stroke="#FF1E42"
            strokeLinecap="round"
            className="animate-pulse"
            style={{ animationDelay: '0.15s' }}
          />
        </svg>
      </div>
    );
  }

  return (
    <div className="flex h-8 w-full items-center justify-center select-none pointer-events-none">
      <svg width="2" height="32" viewBox="0 0 2 32" fill="none" xmlns="http://www.w3.org/2000/svg">
        <rect width="2" height="32" fill="#CBD5E1" />
      </svg>
    </div>
  );
});

export default RouteWireSegment;
