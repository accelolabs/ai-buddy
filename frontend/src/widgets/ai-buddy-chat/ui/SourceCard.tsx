import type { WikiSourcePayload } from '@/entities/chat';

export interface SourceCardProps {
  source: WikiSourcePayload;
}

const DOCUMENT_ICON = (
  <svg
    className="h-4 w-4 shrink-0 text-[#00AAFF]"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
    <polyline points="14 2 14 8 20 8" />
    <line x1="16" y1="13" x2="8" y2="13" />
    <line x1="16" y1="17" x2="8" y2="17" />
    <polyline points="10 9 9 9 8 9" />
  </svg>
);

export function SourceCard({ source }: SourceCardProps) {
  const content = (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          {DOCUMENT_ICON}
          <span className="text-sm font-semibold text-[#00AAFF] hover:underline">
            {source.title}
          </span>
        </div>
        {source.section && (
          <span className="rounded-md bg-[#EDF2F7] px-2.5 py-1 text-xs font-medium text-[#64748B]">
            {source.section}
          </span>
        )}
      </div>

      {source.quote && <p className="text-[13px] leading-relaxed text-[#64748B]">{source.quote}</p>}
    </div>
  );

  return (
    <div className="w-full rounded-[16px] border border-[#BFDBFE] bg-white p-4 transition-shadow duration-200 hover:shadow-xs">
      {source.url ? (
        <a href={source.url} target="_blank" rel="noopener noreferrer" className="block">
          {content}
        </a>
      ) : (
        content
      )}
    </div>
  );
}

export default SourceCard;
