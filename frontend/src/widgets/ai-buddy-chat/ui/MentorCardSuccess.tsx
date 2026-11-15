import { Check, ExternalLink } from 'lucide-react';
import { Link } from 'react-router';
import type { CreatedTicket } from '@/entities/chat/api/ticketApi';

export interface MentorCardSuccessProps {
  ticket: CreatedTicket;
}

export function MentorCardSuccess({ ticket }: MentorCardSuccessProps) {
  return (
    <div
      data-testid="mentor-ticket-success"
      className="flex w-full flex-wrap items-center justify-between gap-3 rounded-[16px] border border-emerald-200 bg-emerald-50/70 p-4 text-[14px] leading-snug text-[#0F172A] shadow-xs transition-all duration-300 animate-in fade-in zoom-in-95"
    >
      <div className="flex items-center gap-2.5">
        <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-emerald-100 ring-2 ring-emerald-200/50">
          <Check className="h-4 w-4 text-emerald-600" strokeWidth={2.5} />
        </div>
        <div>
          Успешно отправлен тикет{' '}
          <strong className="font-semibold text-emerald-950">«{ticket.subject}»</strong>.
        </div>
      </div>

      <Link
        to={`/tickets/${ticket.id}`}
        title={`Перейти к тикету #${ticket.id}`}
        className="inline-flex items-center gap-1.5 rounded-xl bg-white px-3.5 py-1.5 font-bold text-[#0284C7] shadow-xs ring-1 ring-slate-200 transition-all duration-150 hover:bg-[#F0F9FF] hover:text-[#0369A1] hover:ring-[#BAE6FD] active:scale-[0.98]"
      >
        <span>Перейти к тикету #{ticket.id}</span>
        <ExternalLink className="h-3.5 w-3.5 inline-block shrink-0" strokeWidth={2.2} />
      </Link>
    </div>
  );
}
