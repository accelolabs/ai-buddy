import { memo } from 'react';
import { Link as LinkIcon, Check, User as UserIcon } from 'lucide-react';
import { useShallow } from 'zustand/react/shallow';
import type { Ticket } from '@/entities/ticket';
import { formatDateFull } from '../lib/formatDate';
import { CIRCUIT_ANIMATION_STYLES } from '../constants/styles';
import { DetailCircuit } from './circuits/DetailCircuit';
import { CornerBrackets } from './circuits/CornerBrackets';
import { TicketStatusBadge } from './TicketStatusBadge';
import { useTicketsUiStore } from '../store';

export interface TicketDetailViewProps {
  ticket: Ticket | null;
}

export const TicketDetailView = memo(function TicketDetailView({ ticket }: TicketDetailViewProps) {
  const { copiedTicketId, copyTicketLink } = useTicketsUiStore(
    useShallow((state) => ({
      copiedTicketId: state.copiedTicketId,
      copyTicketLink: state.copyTicketLink,
    }))
  );

  if (!ticket) {
    return (
      <div className="flex h-[580px] w-full items-center justify-center rounded-[16px] border border-slate-200/90 bg-white p-6 text-sm text-slate-400 shadow-xs">
        Выберите тикет для просмотра
      </div>
    );
  }

  const isCopied = copiedTicketId === ticket.id;
  const ticketUrl = ticket.url || `https://tickets.ai-buddy.local/t/${ticket.id}`;

  const handleCopyLink = () => {
    void copyTicketLink(ticket.id, ticketUrl);
  };

  return (
    <article className="relative flex flex-col justify-between overflow-hidden rounded-[16px] border border-slate-200/90 bg-white p-6 shadow-xs sm:p-7 min-h-[584px]">
      <style>{CIRCUIT_ANIMATION_STYLES}</style>

      <CornerBrackets />
      <DetailCircuit />

      <div className="relative z-10 flex flex-col gap-5">
        <div className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-100 pb-3">
          <div className="flex items-center gap-2">
            <span className="font-semibold text-slate-900 text-sm">{ticket.id}</span>
            <span className="text-slate-300">·</span>
            <TicketStatusBadge status={ticket.status} showIcon={true} />
          </div>
          <time className="text-xs text-slate-400">{formatDateFull(ticket.createdAt)}</time>
        </div>

        <h1 className="text-xl font-bold leading-tight text-slate-900 sm:text-2xl">
          {ticket.subject}
        </h1>

        <div className="flex flex-col gap-2 rounded-xl border border-slate-200/90 bg-slate-50/60 p-2 sm:flex-row sm:items-center sm:justify-between sm:p-2.5">
          <div className="flex min-w-0 items-center gap-2 px-1">
            <LinkIcon className="h-4 w-4 shrink-0 text-sky-500" />
            <span className="truncate text-xs font-medium text-[#0284C7] sm:text-sm">
              {ticketUrl}
            </span>
          </div>

          <button
            type="button"
            onClick={handleCopyLink}
            className="inline-flex h-8 shrink-0 items-center justify-center gap-1.5 rounded-[8px] border border-slate-200 bg-white px-3 text-xs font-medium text-slate-700 shadow-2xs transition-colors hover:bg-slate-50 hover:text-slate-900 active:scale-[0.98]"
          >
            {isCopied ? (
              <>
                <Check className="h-3.5 w-3.5 text-emerald-600" />
                <span className="text-emerald-700 font-semibold">Скопировано!</span>
              </>
            ) : (
              <span>Копировать ссылку</span>
            )}
          </button>
        </div>

        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div className="flex flex-col gap-1.5 rounded-xl border border-slate-200/90 bg-slate-50/40 p-3 sm:p-3.5">
            <span className="text-[11px] font-semibold uppercase tracking-wider text-slate-400">
              ОТПРАВИТЕЛЬ
            </span>
            <div className="flex items-center gap-2.5">
              <div className="flex flex-col gap-0.5 text-slate-300">
                <span className="h-0.5 w-3.5 rounded-full bg-slate-300" />
                <span className="h-0.5 w-3.5 rounded-full bg-slate-300" />
                <span className="h-0.5 w-3.5 rounded-full bg-slate-300" />
              </div>
              <div className="flex min-w-0 flex-col">
                <span className="truncate font-bold text-slate-900 text-sm">
                  {ticket.author.name}
                </span>
                <span className="truncate text-xs text-slate-500">
                  {ticket.author.role}
                  {ticket.author.department ? ` · ${ticket.author.department}` : ''}
                </span>
              </div>
            </div>
          </div>

          <div className="flex flex-col gap-1.5 rounded-xl border border-slate-200/90 bg-slate-50/40 p-3 sm:p-3.5">
            <span className="text-[11px] font-semibold uppercase tracking-wider text-slate-400">
              ПОЛУЧАТЕЛЬ
            </span>
            <div className="flex items-center gap-2.5">
              <div className="flex flex-col gap-0.5 text-slate-300">
                <span className="h-0.5 w-3.5 rounded-full bg-slate-300" />
                <span className="h-0.5 w-3.5 rounded-full bg-slate-300" />
                <span className="h-0.5 w-3.5 rounded-full bg-slate-300" />
              </div>
              <div className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-sky-100 text-sky-600">
                <UserIcon className="h-3 w-3" />
              </div>
              <div className="flex min-w-0 flex-col">
                <span className="truncate font-bold text-slate-900 text-sm">
                  {ticket.recipient.name}
                </span>
                <span className="truncate text-xs text-slate-500">
                  {ticket.recipient.role}
                </span>
              </div>
            </div>
          </div>
        </div>

        <div className="flex flex-col gap-1.5">
          <span className="text-[11px] font-semibold uppercase tracking-wider text-slate-400">
            ОПИСАНИЕ ОБРАЩЕНИЯ
          </span>
          <div className="rounded-xl border border-slate-200/90 bg-white/95 p-4 text-xs leading-relaxed text-slate-700 sm:text-sm">
            {ticket.description}
          </div>
        </div>
      </div>
    </article>
  );
});

export default TicketDetailView;
