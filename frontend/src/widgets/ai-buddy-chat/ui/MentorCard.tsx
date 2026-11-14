import type { MentorPayload } from '@/entities/chat';

export interface MentorCardProps {
  mentor: MentorPayload;
  onContact?: (mentor: MentorPayload) => void;
}

const MAIL_ICON = (
  <svg
    className="h-4 w-4 text-white"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z" />
    <polyline points="22,6 12,13 2,6" />
  </svg>
);

export function MentorCard({ mentor, onContact }: MentorCardProps) {
  return (
    <div className="flex w-full flex-wrap items-center justify-between gap-4 rounded-[16px] border border-slate-200 bg-white p-4 shadow-xs">
      <div className="flex items-center gap-3.5">
        <div className="relative flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-[#89C6FC] text-sm font-bold text-[#004B87]">
          {mentor.initials}
          {mentor.isOnline && (
            <span
              className="absolute -bottom-0.5 -right-0.5 h-3.5 w-3.5 rounded-full border-2 border-white bg-[#04E061]"
              title="В сети"
            />
          )}
        </div>

        <div className="flex flex-col gap-0.5">
          <span className="text-sm font-bold text-[#0F172A]">{mentor.name}</span>
          <span className="text-xs text-[#64748B]">
            {mentor.role} · {mentor.team}
          </span>
          <span className="text-xs text-[#64748B]">{mentor.schedule}</span>
        </div>
      </div>

      <button
        type="button"
        onClick={() => onContact?.(mentor)}
        className="flex items-center gap-2 rounded-xl bg-[#00AAFF] px-4 py-2 text-sm font-medium text-white transition-all duration-200 hover:bg-[#0096E6] hover:shadow-xs active:scale-[0.98]"
      >
        {MAIL_ICON}
        <span>Сообщить</span>
      </button>
    </div>
  );
}

export default MentorCard;
