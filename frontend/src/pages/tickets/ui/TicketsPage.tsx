import { useEffect } from 'react';
import { useParams, useNavigate } from 'react-router';
import { useTickets, type Ticket } from '@/entities/ticket';
import { TicketsHeader, TicketDetailView, TicketsList } from '@/widgets/tickets';

export function TicketsPage() {
  const { ticketId } = useParams();
  const navigate = useNavigate();
  const { data: tickets = [], isLoading, isError } = useTickets();

  const selectedTicket: Ticket | null =
    (ticketId
      ? tickets.find((t) => t.id.toLowerCase() === ticketId.toLowerCase())
      : null) ||
    tickets[0] ||
    null;

  useEffect(() => {
    if (!ticketId && tickets.length > 0) {
      navigate(`/tickets/${tickets[0].id}`, { replace: true });
    }
  }, [ticketId, tickets, navigate]);

  const handleSelectTicket = (ticket: Ticket) => {
    navigate(`/tickets/${ticket.id}`);
  };

  return (
    <div className="min-h-screen w-full bg-[#F8FAFC] p-4 sm:p-6 lg:p-8">
      <div className="mx-auto max-w-6xl">
        <TicketsHeader />

        {isLoading ? (
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
            <div className="h-[580px] animate-pulse rounded-[16px] bg-slate-200/60 lg:col-span-7" />
            <div className="flex flex-col gap-3 lg:col-span-5">
              <div className="h-6 w-36 animate-pulse rounded bg-slate-200" />
              <div className="h-28 animate-pulse rounded-[16px] bg-slate-200/60" />
              <div className="h-28 animate-pulse rounded-[16px] bg-slate-200/60" />
              <div className="h-28 animate-pulse rounded-[16px] bg-slate-200/60" />
            </div>
          </div>
        ) : isError ? (
          <div className="flex h-64 flex-col items-center justify-center rounded-[16px] border border-red-200 bg-red-50 p-6 text-center text-sm text-red-600">
            <p className="font-semibold">Не удалось загрузить тикеты</p>
            <p className="mt-1 text-xs text-red-500">Попробуйте обновить страницу</p>
          </div>
        ) : (
          <main className="grid grid-cols-1 items-start gap-6 lg:grid-cols-12">
            <div className="lg:col-span-7">
              <TicketDetailView ticket={selectedTicket} />
            </div>
            <div className="lg:col-span-5">
              <TicketsList
                tickets={tickets}
                activeTicketId={selectedTicket?.id}
                onSelectTicket={handleSelectTicket}
              />
            </div>
          </main>
        )}
      </div>
    </div>
  );
}

export default TicketsPage;
