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

export async function getTicketById(id: string): Promise<Ticket | null> {
  try {
    const response = await fetch(`/api/tickets/${id}`, {
      headers: {
        Accept: 'application/json',
      },
    });

    if (response.ok) {
      const data = await response.json();
      if (data && data.id) {
        return data as Ticket;
      }
    }
  } catch {
    void 0;
  }

  const allTickets = await getTickets();
  const found = allTickets.find(
    (t) => t.id.toLowerCase() === id.toLowerCase(),
  );

  return found || null;
}
