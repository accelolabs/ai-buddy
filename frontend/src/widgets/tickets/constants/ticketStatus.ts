import type { TicketStatus } from '@/entities/ticket';

export interface TicketStatusMeta {
  label: string;
  dotColorClass: string;
  textColorClass: string;
  badgeBgClass: string;
}

export const TICKET_STATUS_MAP: Record<TicketStatus, TicketStatusMeta> = {
  in_review: {
    label: 'На рассмотрении',
    dotColorClass: 'bg-amber-500',
    textColorClass: 'text-amber-600',
    badgeBgClass: 'bg-amber-50 text-amber-700 border-amber-200',
  },
  resolved: {
    label: 'Ответ получен',
    dotColorClass: 'bg-emerald-500',
    textColorClass: 'text-emerald-600',
    badgeBgClass: 'bg-emerald-50 text-emerald-700 border-emerald-200',
  },
};
