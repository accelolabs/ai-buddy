import { useQuery } from '@tanstack/react-query';
import type { Ticket } from '../model/types';
import { getTickets } from './ticketApi';

export function useTickets() {
  return useQuery<Ticket[]>({
    queryKey: ['tickets'],
    queryFn: getTickets,
  });
}
