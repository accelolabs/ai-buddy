import type { Ticket } from '../model/types';
import ticketsData from '../../../../../mock-server/tickets.json';

export async function getTickets(): Promise<Ticket[]> {
  try {
    const response = await fetch('/api/tickets', {
      headers: {
        Accept: 'application/json',
      },
    });

    if (response.ok) {
      const data = await response.json();
      if (Array.isArray(data)) {
        return data as Ticket[];
      }
      if (data && Array.isArray(data.tickets)) {
        return data.tickets as Ticket[];
      }
    }
  } catch {
    void 0;
  }

  return (ticketsData.tickets || []) as Ticket[];
}
