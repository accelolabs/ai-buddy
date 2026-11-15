import { useQuery } from '@tanstack/react-query';
import type { Ticket } from '../model/types';
import { getTickets, getTicketById } from './ticketApi';

export function useTickets() {
  return useQuery<Ticket[]>({
    queryKey: ['tickets'],
    queryFn: getTickets,
  });
}

export function useTicket(id?: string) {
  return useQuery<Ticket | null>({
    queryKey: ['tickets', id],
    queryFn: () => (id ? getTicketById(id) : Promise.resolve(null)),
    enabled: Boolean(id),
  });
}
