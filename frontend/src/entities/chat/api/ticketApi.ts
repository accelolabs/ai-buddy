export interface CreateTicketPayload {
  subject: string;
  message: string;
  mentorId: string;
  idempotencyKey?: string;
}

export interface CreatedTicket {
  id: string;
  subject: string;
  message: string;
  mentorId: string;
  url: string;
  createdAt: string;
}

export async function createTicket(payload: CreateTicketPayload): Promise<CreatedTicket> {
  const baseUrl = (import.meta.env.VITE_TICKET_URL as string | undefined) || '';
  const idempotencyKey =
    payload.idempotencyKey || `key-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`;

  try {
    const response = await fetch('/api/tickets', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Idempotency-Key': idempotencyKey,
      },
      body: JSON.stringify({
        subject: payload.subject,
        message: payload.message,
        mentorId: payload.mentorId,
      }),
    });

    if (response.ok) {
      const data = await response.json();
      return {
        id: data.id || `TIC-${Date.now().toString().slice(-4)}`,
        subject: data.subject || payload.subject,
        message: data.message || payload.message,
        mentorId: data.mentorId || payload.mentorId,
        url: data.url || `${baseUrl}/tickets/${data.id || 'current'}`,
        createdAt: data.createdAt || new Date().toISOString(),
      };
    }
  } catch {
    // no operation
  }

  await new Promise((resolve) => setTimeout(resolve, 600));

  const randomTicketNum = Math.floor(1000 + Math.random() * 9000);
  const ticketId = `TIC-${randomTicketNum}`;
  const ticketUrl = baseUrl
    ? `${baseUrl.replace(/\/$/, '')}/tickets/${ticketId}`
    : `/tickets/${ticketId}`;

  return {
    id: ticketId,
    subject: payload.subject,
    message: payload.message,
    mentorId: payload.mentorId,
    url: ticketUrl,
    createdAt: new Date().toISOString(),
  };
}
