import type { ComponentProps } from 'react';

const CORNER_SVG = (
  <svg
    aria-hidden="true"
    className="absolute left-0 top-0 h-7 w-7 text-slate-200 transition-colors duration-300 group-hover:text-blue-300"
    viewBox="0 0 28 28"
    fill="none"
  >
    <path d="M0 0 L18 0 L0 18 Z" fill="white" />
    <path d="M0 18 L18 0" stroke="currentColor" strokeWidth="1.5" />
    <circle
      cx="5.5"
      cy="5.5"
      r="1.5"
      fill="currentColor"
      className="transition-all duration-300 group-hover:animate-pulse group-hover:fill-blue-400 group-hover:drop-shadow-[0_0_6px_rgba(59,130,246,0.8)]"
    />
  </svg>
);

const ARROW_SVG = (
  <svg
    aria-hidden="true"
    className="absolute right-5 top-5 h-4 w-4 text-slate-400 transition-transform duration-300 group-hover:-translate-y-0.5 group-hover:translate-x-0.5 group-hover:text-blue-500 group-hover:drop-shadow-[0_0_4px_rgba(59,130,246,0.4)]"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="1.5"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <path d="M7 17L17 7M7 7h10v10" />
  </svg>
);

const GRID_SVG = (
  <svg
    aria-hidden="true"
    className="absolute bottom-0 right-0 h-16 w-[180px] pointer-events-none"
    viewBox="0 0 180 64"
    fill="none"
  >
    <g
      className="stroke-slate-200 transition-all duration-300 group-hover:stroke-blue-300"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <circle
        cx="100"
        cy="38"
        r="2"
        className="fill-slate-200 transition-all duration-300 group-hover:fill-blue-400 group-hover:drop-shadow-[0_0_6px_rgba(96,165,250,0.8)]"
        stroke="none"
      />
      <path
        d="M 100 38 L 125 38 L 140 23 L 180 23"
        className="transition-all duration-300 group-hover:drop-shadow-[0_0_4px_rgba(147,197,253,0.5)]"
      />

      <circle
        cx="115"
        cy="48"
        r="2"
        className="fill-slate-200 transition-all duration-300 group-hover:fill-blue-400 group-hover:drop-shadow-[0_0_6px_rgba(96,165,250,0.8)]"
        stroke="none"
      />
      <path
        d="M 115 48 L 132 48 L 145 35 L 180 35"
        className="transition-all duration-300 group-hover:drop-shadow-[0_0_4px_rgba(147,197,253,0.5)]"
      />

      <path
        d="M 160 64 L 160 50 L 170 40 L 180 40"
        className="transition-all duration-300 group-hover:drop-shadow-[0_0_4px_rgba(147,197,253,0.5)]"
      />
    </g>

    <g
      className="animate-current stroke-blue-500 opacity-0 transition-opacity duration-300 group-hover:opacity-100 drop-shadow-[0_0_8px_rgba(59,130,246,0.9)]"
      strokeWidth="1.5"
      strokeDasharray="15 35"
      strokeLinecap="round"
    >
      <path d="M 100 38 L 125 38 L 140 23 L 180 23" />
    </g>
  </svg>
);

export interface PromptButtonProps extends ComponentProps<'button'> {
  title: string;
  subtitle: string;
}

export const PromptButton = ({
  title,
  subtitle,
  className = '',
  type = 'button',
  ...props
}: PromptButtonProps) => {
  return (
    <button
      type={type}
      className={`group relative flex w-full max-w-[400px] flex-col justify-center overflow-hidden rounded-[20px] border border-slate-200 bg-white px-6 py-5 text-left transition-colors duration-300 hover:border-blue-400 hover:shadow-[0_0_15px_rgba(59,130,246,0.1)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:border-transparent ${className}`}
      {...props}
    >
      {CORNER_SVG}

      <div className="relative z-10 pr-6">
        <h3 className="text-[15px] font-semibold tracking-tight text-[#161C2D]">{title}</h3>
        <p className="mt-1 text-[13px] text-slate-500">{subtitle}</p>
      </div>

      {ARROW_SVG}
      {GRID_SVG}
    </button>
  );
};
