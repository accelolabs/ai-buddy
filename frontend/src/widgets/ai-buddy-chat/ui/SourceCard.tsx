import { memo } from 'react';
import { FileText } from 'lucide-react';
import type { WikiSourcePayload } from '@/entities/chat';

export interface SourceCardProps {
  source: WikiSourcePayload;
}

export const SourceCard = memo(function SourceCard({ source }: SourceCardProps) {
  const content = (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <FileText className="h-4 w-4 shrink-0 text-[#00AAFF]" />
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
});

export default SourceCard;
