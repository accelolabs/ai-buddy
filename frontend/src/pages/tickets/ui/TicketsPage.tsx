import { Link, useParams } from 'react-router';
import { useQuery } from '@tanstack/react-query';
import { request, ApiError } from '@/shared/api';
import type { Ticket } from '@/shared/api/contracts';
import { dateTime } from '@/shared/api/contracts';
import { CardCircuit } from '@/widgets/tickets/ui/circuits/CardCircuit';
import { DetailCircuit } from '@/widgets/tickets/ui/circuits/DetailCircuit';
import { CornerBrackets } from '@/widgets/tickets/ui/circuits/CornerBrackets';
import { CIRCUIT_ANIMATION_STYLES } from '@/widgets/tickets/constants/styles';
import { RequestState } from '@/shared/ui/RequestState';

export function TicketsPage() {
  const { ticketId } = useParams();
  const tickets = useQuery({
    queryKey: ['tickets'],
    queryFn: ({ signal }) => request<Ticket[]>('/api/tickets', { signal }),
  });
  const detail = useQuery({
    queryKey: ['tickets', ticketId],
    queryFn: ({ signal }) =>
      request<Ticket>(`/api/tickets/${encodeURIComponent(ticketId!)}`, { signal }),
    enabled: !!ticketId,
    retry: false,
  });
  const selected = ticketId ? detail.data : tickets.data?.[0];
  return (
    <div className="min-h-full bg-[#F8FAFC] p-4 sm:p-6 lg:p-8">
      <div className="mx-auto max-w-6xl">
        <style>{CIRCUIT_ANIMATION_STYLES}</style>
        <h1 className="mb-6 text-3xl font-bold text-slate-900">Обращения и тикеты</h1>
        <RequestState
          loading={tickets.isPending}
          error={tickets.error}
          retry={() => void tickets.refetch()}
          empty={tickets.isSuccess && !tickets.data.length ? 'Тикетов пока нет' : undefined}
        />
        <main className="grid items-start gap-6 lg:grid-cols-12">
          <div className="lg:col-span-7">
            {ticketId &&
              (detail.error instanceof ApiError && detail.error.status === 404 ? (
                <p className="p-6">Тикет не найден</p>
              ) : (
                <RequestState
                  loading={detail.isPending}
                  error={detail.error}
                  retry={() => void detail.refetch()}
                />
              ))}
            {selected && (
              <article className="relative min-h-[584px] overflow-hidden rounded-2xl border border-slate-200 bg-white p-6 shadow-xs">
                <CornerBrackets />
                <DetailCircuit />
                <div className="relative z-10">
                  <p className="break-all text-xs text-sky-700">Тикет {selected.id}</p>
                  <h2 className="my-3 text-xl font-bold">{selected.title}</h2>
                  <p className="text-xs text-slate-500">
                    {dateTime(selected.created_at)} · Автор: {selected.user_id}
                  </p>
                  <p className="mt-6 whitespace-pre-wrap break-words text-slate-700">
                    {selected.description}
                  </p>
                </div>
              </article>
            )}
          </div>
          <div className="space-y-3 lg:col-span-5">
            {tickets.data?.map((ticket, index) => (
              <Link
                key={ticket.id}
                to={`/tickets/${encodeURIComponent(ticket.id)}`}
                className={`relative block overflow-hidden rounded-2xl border bg-white p-5 shadow-xs ${selected?.id === ticket.id ? 'border-sky-300' : 'border-slate-200'}`}
              >
                <CardCircuit
                  index={index}
                  ticketId={ticket.id}
                  isActive={selected?.id === ticket.id}
                />
                <div className="relative z-10">
                  <h3 className="font-bold">{ticket.title}</h3>
                  <p className="mt-2 text-xs text-slate-500">{dateTime(ticket.created_at)}</p>
                  <p className="mt-2 line-clamp-2 text-sm text-slate-600">{ticket.description}</p>
                </div>
              </Link>
            ))}
          </div>
        </main>
      </div>
    </div>
  );
}
