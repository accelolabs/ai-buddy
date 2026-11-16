export type TicketStatus = 'in_review' | 'resolved';

export interface TicketParticipant {
  name: string;
  role: string;
  department?: string;
}

export interface Ticket {
  id: string;
  subject: string;
  description: string;
  status: TicketStatus;
  createdAt: string;
  author: TicketParticipant;
  recipient: TicketParticipant;
  url: string;
}
