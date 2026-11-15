import { Link } from 'react-router';
import type { CreatedTicketPayload } from '@/entities/chat';

export interface TicketSuccessCardProps {
  ticket: CreatedTicketPayload;
}

const CHECK_ICON = (
  <svg
    className="h-4 w-4 text-emerald-600"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2.5"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <polyline points="20 6 9 17 4 12" />
  </svg>
);

const EXTERNAL_LINK_ICON = (
  <svg
    className="h-3.5 w-3.5 shrink-0"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2.2"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" />
    <polyline points="15 3 21 3 21 9" />
    <line x1="10" y1="14" x2="21" y2="3" />
  </svg>
);

export function TicketSuccessCard({ ticket }: TicketSuccessCardProps) {
  return (
    <div
      data-testid="ticket-success-card"
      className="flex w-full flex-col gap-3 rounded-[16px] border border-emerald-200 bg-emerald-50/70 p-4 text-[14px] leading-snug text-[#0F172A] shadow-xs transition-all duration-200 animate-in fade-in zoom-in-95"
    >
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2.5">
          <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-emerald-100 ring-2 ring-emerald-200/50">
            {CHECK_ICON}
          </div>
          <div>
            Успешно отправлен тикет{' '}
            <strong className="font-semibold text-emerald-950">«{ticket.subject}»</strong>.
          </div>
        </div>

        <Link
          to={`/tickets/${ticket.id}`}
          title={`Перейти к тикету #${ticket.id}`}
          className="inline-flex items-center gap-1.5 rounded-xl bg-white px-3.5 py-1.5 text-xs font-bold text-[#0284C7] shadow-xs ring-1 ring-slate-200 transition-all duration-150 hover:bg-[#F0F9FF] hover:text-[#0369A1] hover:ring-[#BAE6FD] active:scale-[0.98]"
        >
          <span>Перейти к тикету #{ticket.id}</span>
          {EXTERNAL_LINK_ICON}
        </Link>
      </div>
    </div>
  );
}

export default TicketSuccessCard;
