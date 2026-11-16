import { memo } from 'react';
import { ArrowUpRight } from 'lucide-react';
import type { Ticket } from '@/entities/ticket';
import { formatDateShort } from '../lib/formatDate';
import { CIRCUIT_ANIMATION_STYLES } from '../constants/styles';
import { CardCircuit } from './circuits/CardCircuit';
import { TicketStatusBadge } from './TicketStatusBadge';

export interface TicketsListProps {
  tickets: Ticket[];
  activeTicketId?: string;
  onSelectTicket: (ticket: Ticket) => void;
}

interface TicketCardItemProps {
  ticket: Ticket;
  index: number;
  isActive: boolean;
  onSelect: (ticket: Ticket) => void;
}

const TicketCardItem = memo(function TicketCardItem({
  ticket,
  index,
  isActive,
  onSelect,
}: TicketCardItemProps) {
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={() => onSelect(ticket)}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          onSelect(ticket);
        }
      }}
      className={`group relative flex flex-col justify-between overflow-hidden rounded-[16px] border p-4 text-left transition-all duration-150 cursor-pointer min-h-[97px] ${
        isActive
          ? 'border-sky-400 bg-white ring-2 ring-sky-400/20 shadow-xs'
          : 'border-slate-200/90 bg-white hover:border-slate-300 hover:shadow-2xs'
      }`}
    >
      <CardCircuit index={index} isActive={isActive} ticketId={ticket.id} />

      <div className="relative z-10 flex flex-col gap-1.5">
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-1.5">
            <TicketStatusBadge status={ticket.status} />
            <span className="text-slate-300">·</span>
            <span className="font-bold text-xs text-slate-900">{ticket.id}</span>
          </div>

          <ArrowUpRight
            className={`h-4 w-4 shrink-0 transition-colors ${
              isActive ? 'text-sky-500' : 'text-slate-400 group-hover:text-slate-600'
            }`}
          />
        </div>

        <div className="text-sm font-bold text-slate-900 line-clamp-2">{ticket.subject}</div>

        <div className="mt-1 text-xs text-slate-500">
          <span>Наставник: {ticket.recipient.name}</span>
          <span className="mx-1.5 text-slate-300">·</span>
          <time>{formatDateShort(ticket.createdAt)}</time>
        </div>
      </div>
    </div>
  );
});

export function TicketsList({ tickets, activeTicketId, onSelectTicket }: TicketsListProps) {
  return (
    <section className="flex flex-col gap-3">
      <style>{CIRCUIT_ANIMATION_STYLES}</style>

      <h2 className="text-base font-bold text-slate-900">Список тикетов ({tickets.length})</h2>

      <div className="flex flex-col gap-3">
        {tickets.map((ticket, index) => {
          const isActive = ticket.id.toLowerCase() === (activeTicketId || '').toLowerCase();
          return (
            <TicketCardItem
              key={ticket.id}
              ticket={ticket}
              index={index}
              isActive={isActive}
              onSelect={onSelectTicket}
            />
          );
        })}
      </div>
    </section>
  );
}

export default TicketsList;
