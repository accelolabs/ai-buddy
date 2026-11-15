import { memo } from 'react';
import { Clock, CheckCircle2 } from 'lucide-react';
import type { TicketStatus } from '@/entities/ticket';
import { TICKET_STATUS_MAP } from '../constants/ticketStatus';

export interface TicketStatusBadgeProps {
  status: TicketStatus;
  showIcon?: boolean;
}

export const TicketStatusBadge = memo(function TicketStatusBadge({
  status,
  showIcon = false,
}: TicketStatusBadgeProps) {
  const meta = TICKET_STATUS_MAP[status] || TICKET_STATUS_MAP.in_review;

  if (showIcon) {
    return (
      <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${meta.textColorClass}`}>
        {status === 'in_review' ? (
          <Clock className="h-3.5 w-3.5 text-amber-500 shrink-0" />
        ) : (
          <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500 shrink-0" />
        )}
        <span>{meta.label}</span>
      </span>
    );
  }

  return (
    <div className="flex items-center gap-1.5">
      <span className={`h-2 w-2 rounded-full ${meta.dotColorClass}`} />
      <span className={`text-xs font-medium ${meta.textColorClass}`}>{meta.label}</span>
    </div>
  );
});
