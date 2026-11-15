import { ArrowUpRight } from 'lucide-react';
import type { Ticket } from '@/entities/ticket';

export interface TicketsListProps {
  tickets: Ticket[];
  activeTicketId?: string;
  onSelectTicket: (ticket: Ticket) => void;
}

function formatDateShort(iso: string): string {
  try {
    const d = new Date(iso);
    const day = String(d.getDate()).padStart(2, '0');
    const month = String(d.getMonth() + 1).padStart(2, '0');
    const year = d.getFullYear();
    return `${day}.${month}.${year}`;
  } catch {
    return iso;
  }
}

function CardCircuit({
  index,
  isActive,
  ticketId,
}: {
  index: number;
  isActive: boolean;
  ticketId: string;
}) {
  const patternIndex = index % 3;

  if (patternIndex === 0) {
    return (
      <svg
        viewBox="639 140 425 97.31"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className="pointer-events-none absolute inset-0 h-full w-full"
        preserveAspectRatio="none"
      >
        <defs>
          <filter id={`cardGlow-${ticketId}`} x="-30%" y="-30%" width="160%" height="160%">
            <feGaussianBlur stdDeviation="2.5" result="blur" />
            <feMerge>
              <feMergeNode in="blur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>
        </defs>

        <path d="M640 152H647L651 148V141" stroke="#38BDF8" strokeWidth="1.2" />
        <circle cx="645" cy="146" r="1.4" fill="#0EA5E9" />

        <g stroke={isActive ? '#BAE6FD' : '#E2E8F0'} strokeWidth="1.2" strokeLinecap="round">
          <path d="M1063 198.31H1025L1007 216.31H975" />
          <path d="M1063 207.31H1029L1012 224.31H991" />
          <path d="M1035 236.31V223.31L1046 212.31H1063" />
          <path d="M1063 198.31H1061H1059H1057H1047" stroke="#0EA5E9" strokeWidth="1.6" />
        </g>

        <g
          style={{ color: isActive ? '#0EA5E9' : '#94A3B8' }}
          className={isActive ? 'animate-neon' : 'group-hover:animate-neon'}
        >
          <path
            d="M1063 198.31H1025L1007 216.31H975"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinecap="round"
            pathLength="1"
            className="animate-flow"
            style={{ animationDuration: '3.2s' }}
          />
          <path
            d="M1063 207.31H1029L1012 224.31H991"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinecap="round"
            pathLength="1"
            className="animate-flow"
            style={{ animationDuration: '3.6s', animationDelay: '0.8s' }}
          />
          <path
            d="M1035 236.31V223.31L1046 212.31H1063"
            stroke="currentColor"
            strokeWidth="1.5"
            strokeLinecap="round"
            pathLength="1"
            className="animate-flow"
            style={{ animationDuration: '3.4s', animationDelay: '0.4s' }}
          />

          <circle cx="975" cy="216.31" r="5" fill="currentColor" opacity="0.35" filter={`url(#cardGlow-${ticketId})`} />
          <circle cx="975" cy="216.31" r="3" fill="currentColor" />
          <circle cx="975" cy="216.31" r="1.2" fill="#FFFFFF" opacity="0.9" />

          <circle cx="991" cy="224.31" r="5" fill="currentColor" opacity="0.35" filter={`url(#cardGlow-${ticketId})`} />
          <circle cx="991" cy="224.31" r="3" fill="currentColor" />
          <circle cx="991" cy="224.31" r="1.2" fill="#FFFFFF" opacity="0.9" />

          <circle cx="1046" cy="212.31" r="5" fill="currentColor" opacity="0.35" filter={`url(#cardGlow-${ticketId})`} />
          <circle cx="1046" cy="212.31" r="3" fill="currentColor" />
          <circle cx="1046" cy="212.31" r="1.2" fill="#FFFFFF" opacity="0.9" />
        </g>
      </svg>
    );
  }

  if (patternIndex === 1) {
    return (
      <svg
        viewBox="639 249.31 425 97.31"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className="pointer-events-none absolute inset-0 h-full w-full"
        preserveAspectRatio="none"
      >
        <defs>
          <filter id={`cardGlow-${ticketId}`} x="-30%" y="-30%" width="160%" height="160%">
            <feGaussianBlur stdDeviation="2.5" result="blur" />
            <feMerge>
              <feMergeNode in="blur" />
              <feMergeNode in="SourceGraphic" />
            </feMerge>
          </filter>
        </defs>

        <path d="M640 334.62H647L651 338.62V345.62" stroke="#CBD5E1" strokeWidth="1.1" />
        <circle cx="645" cy="340.62" r="1.4" fill="#94A3B8" />

        <g stroke={isActive ? '#BAE6FD' : '#E2E8F0'} strokeWidth="1.1" strokeLinecap="round">
          <path d="M1063 305.62H1031L1013 323.62H981" />
          <path d="M1063 314.62H1035L1019 330.62V345.62" />
          <path d="M965 345.62L977 333.62H1003" />
          <path d="M1063 305.62H1061H1059H1057H1049H1047" stroke={isActive ? '#0EA5E9' : '#64748B'} strokeWidth="1.4" />
        </g>

        <g
          style={{ color: isActive ? '#0EA5E9' : '#94A3B8' }}
          className={isActive ? 'animate-neon' : 'group-hover:animate-neon'}
        >
          <path
            d="M1063 305.62H1031L1013 323.62H981"
            stroke="currentColor"
            strokeWidth="1.4"
            strokeLinecap="round"
            pathLength="1"
            className="animate-flow"
            style={{ animationDuration: '3.4s' }}
          />
          <path
            d="M1063 314.62H1035L1019 330.62V345.62"
            stroke="currentColor"
            strokeWidth="1.4"
            strokeLinecap="round"
            pathLength="1"
            className="animate-flow"
            style={{ animationDuration: '3.8s', animationDelay: '0.9s' }}
          />
          <path
            d="M965 345.62L977 333.62H1003"
            stroke="currentColor"
            strokeWidth="1.4"
            strokeLinecap="round"
            pathLength="1"
            className="animate-flow"
            style={{ animationDuration: '3.2s', animationDelay: '0.4s' }}
          />

          <circle cx="981" cy="323.62" r="5" fill="currentColor" opacity="0.35" filter={`url(#cardGlow-${ticketId})`} />
          <circle cx="981" cy="323.62" r="3" fill="currentColor" />
          <circle cx="981" cy="323.62" r="1.2" fill="#FFFFFF" opacity="0.9" />

          <circle cx="1003" cy="333.62" r="5" fill="currentColor" opacity="0.35" filter={`url(#cardGlow-${ticketId})`} />
          <circle cx="1003" cy="333.62" r="3" fill="currentColor" />
          <circle cx="1003" cy="333.62" r="1.2" fill="#FFFFFF" opacity="0.9" />
        </g>
      </svg>
    );
  }

  return (
    <svg
      viewBox="639 358.62 425 97.31"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className="pointer-events-none absolute inset-0 h-full w-full"
      preserveAspectRatio="none"
    >
      <defs>
        <filter id={`cardGlow-${ticketId}`} x="-30%" y="-30%" width="160%" height="160%">
          <feGaussianBlur stdDeviation="2.5" result="blur" />
          <feMerge>
            <feMergeNode in="blur" />
            <feMergeNode in="SourceGraphic" />
          </feMerge>
        </filter>
      </defs>

      <path d="M640 370.62H647L651 366.62V359.62" stroke="#CBD5E1" strokeWidth="1.1" />
      <circle cx="645" cy="364.62" r="1.4" fill="#94A3B8" />

      <g stroke={isActive ? '#BAE6FD' : '#E2E8F0'} strokeWidth="1.1" strokeLinecap="round">
        <path d="M1063 415.93H1023L1004 434.93H973" />
        <path d="M991 454.93L1005 440.93H1039L1051 428.93H1063" />
        <path d="M1027 454.93V444.93L1035 436.93" />
        <path d="M1063 415.93H1061H1051H1049H1047" stroke={isActive ? '#0EA5E9' : '#64748B'} strokeWidth="1.4" />
      </g>

      <g
        style={{ color: isActive ? '#0EA5E9' : '#94A3B8' }}
        className={isActive ? 'animate-neon' : 'group-hover:animate-neon'}
      >
        <path
          d="M1063 415.93H1023L1004 434.93H973"
          stroke="currentColor"
          strokeWidth="1.4"
          strokeLinecap="round"
          pathLength="1"
          className="animate-flow"
          style={{ animationDuration: '3.5s' }}
        />
        <path
          d="M991 454.93L1005 440.93H1039L1051 428.93H1063"
          stroke="currentColor"
          strokeWidth="1.4"
          strokeLinecap="round"
          pathLength="1"
          className="animate-flow"
          style={{ animationDuration: '4s', animationDelay: '0.8s' }}
        />
        <path
          d="M1027 454.93V444.93L1035 436.93"
          stroke="currentColor"
          strokeWidth="1.4"
          strokeLinecap="round"
          pathLength="1"
          className="animate-flow"
          style={{ animationDuration: '3s', animationDelay: '0.3s' }}
        />

        <circle cx="973" cy="434.93" r="5" fill="currentColor" opacity="0.35" filter={`url(#cardGlow-${ticketId})`} />
        <circle cx="973" cy="434.93" r="3" fill="currentColor" />
        <circle cx="973" cy="434.93" r="1.2" fill="#FFFFFF" opacity="0.9" />

        <circle cx="1035" cy="436.93" r="5" fill="currentColor" opacity="0.35" filter={`url(#cardGlow-${ticketId})`} />
        <circle cx="1035" cy="436.93" r="3" fill="currentColor" />
        <circle cx="1035" cy="436.93" r="1.2" fill="#FFFFFF" opacity="0.9" />
      </g>
    </svg>
  );
}

export function TicketsList({
  tickets,
  activeTicketId,
  onSelectTicket,
}: TicketsListProps) {
  return (
    <section className="flex flex-col gap-3">
      <style>{`
        @keyframes neon-pulse {
          0%, 100% { opacity: 0.35; filter: drop-shadow(0 0 2px currentColor); }
          50% { opacity: 1; filter: drop-shadow(0 0 6px currentColor); }
        }
        @keyframes dash-flow {
          0% { stroke-dashoffset: 1; }
          100% { stroke-dashoffset: 0; }
        }
        .animate-neon {
          animation: neon-pulse 3s infinite ease-in-out;
        }
        .animate-flow {
          stroke-dasharray: 0.28 0.72;
          animation: dash-flow 4.2s infinite linear;
        }
      `}</style>

      <h2 className="text-base font-bold text-slate-900">
        Список тикетов ({tickets.length})
      </h2>

      <div className="flex flex-col gap-3">
        {tickets.map((ticket, index) => {
          const isActive = ticket.id.toLowerCase() === (activeTicketId || '').toLowerCase();
          const isInReview = ticket.status === 'in_review';

          return (
            <div
              key={ticket.id}
              role="button"
              tabIndex={0}
              onClick={() => onSelectTicket(ticket)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault();
                  onSelectTicket(ticket);
                }
              }}
              className={`group relative flex flex-col justify-between overflow-hidden rounded-[16px] border p-4 text-left transition-all duration-150 cursor-pointer min-h-[97px] ${
                isActive
                  ? 'border-sky-400 bg-white ring-2 ring-sky-400/20 shadow-xs'
                  : 'border-slate-200/90 bg-white hover:border-slate-300 hover:shadow-2xs'
              }`}
            >
              <CardCircuit index={index} isActive={isActive} ticketId={ticket.id} />

              <div className="relative z-10 flex flex-col gap-1.5">
                <div className="flex items-center justify-between gap-2">
                  <div className="flex items-center gap-1.5">
                    <span
                      className={`h-2 w-2 rounded-full ${
                        isInReview ? 'bg-amber-500' : 'bg-emerald-500'
                      }`}
                    />
                    <span className="font-bold text-xs text-slate-900">
                      {ticket.id}
                    </span>
                    <span className="text-slate-300">·</span>
                    <span
                      className={`text-xs font-medium ${
                        isInReview ? 'text-amber-600' : 'text-emerald-600'
                      }`}
                    >
                      {isInReview ? 'На рассмотрении' : 'Ответ получен'}
                    </span>
                  </div>

                  <ArrowUpRight
                    className={`h-4 w-4 shrink-0 transition-colors ${
                      isActive
                        ? 'text-sky-500'
                        : 'text-slate-400 group-hover:text-slate-600'
                    }`}
                  />
                </div>

                <div className="text-sm font-bold text-slate-900 line-clamp-2">
                  {ticket.subject}
                </div>

                <div className="mt-1 text-xs text-slate-500">
                  <span>Наставник: {ticket.recipient.name}</span>
                  <span className="mx-1.5 text-slate-300">·</span>
                  <time>{formatDateShort(ticket.createdAt)}</time>
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </section>
  );
}

export default TicketsList;
