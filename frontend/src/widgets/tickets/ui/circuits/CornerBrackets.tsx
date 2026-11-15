import { memo } from 'react';

export const CornerBrackets = memo(function CornerBrackets() {
  return (
    <>
      <div className="pointer-events-none absolute left-0 top-0 h-4 w-4 z-20">
        <svg viewBox="0 0 16 16" fill="none" className="h-full w-full">
          <path
            d="M0 12V4C0 1.79086 1.79086 0 4 0H12"
            stroke="#7DD3FC"
            strokeWidth="3"
            strokeLinecap="round"
          />
        </svg>
      </div>
      <div className="pointer-events-none absolute right-0 top-0 h-4 w-4 z-20">
        <svg viewBox="0 0 16 16" fill="none" className="h-full w-full">
          <path
            d="M16 12V4C16 1.79086 14.2091 0 12 0H4"
            stroke="#7DD3FC"
            strokeWidth="3"
            strokeLinecap="round"
          />
        </svg>
      </div>
    </>
  );
});
