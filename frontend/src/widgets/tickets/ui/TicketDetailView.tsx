import { useState } from 'react';
import { Clock, CheckCircle2, Link as LinkIcon, Check, User as UserIcon } from 'lucide-react';
import type { Ticket } from '@/entities/ticket';

export interface TicketDetailViewProps {
  ticket: Ticket | null;
}

function formatDate(iso: string): string {
  try {
    const d = new Date(iso);
    const day = String(d.getDate()).padStart(2, '0');
    const month = String(d.getMonth() + 1).padStart(2, '0');
    const year = d.getFullYear();
    const hours = String(d.getHours()).padStart(2, '0');
    const minutes = String(d.getMinutes()).padStart(2, '0');
    return `${day}.${month}.${year}, ${hours}:${minutes}`;
  } catch {
    return iso;
  }
}

export function TicketDetailView({ ticket }: TicketDetailViewProps) {
  const [copied, setCopied] = useState(false);

  if (!ticket) {
    return (
      <div className="flex h-[580px] w-full items-center justify-center rounded-[16px] border border-slate-200/90 bg-white p-6 text-sm text-slate-400 shadow-xs">
        Выберите тикет для просмотра
      </div>
    );
  }

  const ticketUrl =
    ticket.url || `https://tickets.ai-buddy.local/t/${ticket.id}`;

  const handleCopyLink = async () => {
    try {
      await navigator.clipboard.writeText(ticketUrl);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      void 0;
    }
  };

  const isInReview = ticket.status === 'in_review';

  return (
    <article className="relative flex flex-col justify-between overflow-hidden rounded-[16px] border border-slate-200/90 bg-white p-6 shadow-xs sm:p-7 min-h-[584px]">
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

      <div className="pointer-events-none absolute left-0 top-0 h-4 w-4 z-20">
        <svg viewBox="0 0 16 16" fill="none" className="h-full w-full">
          <path
            d="M0 12V4C0 1.79086 1.79086 0 4 0H12"
            stroke="#7DD3FC"
            strokeWidth="3"
            strokeLinecap="round"
          />
        </svg>
      </div>
      <div className="pointer-events-none absolute right-0 top-0 h-4 w-4 z-20">
        <svg viewBox="0 0 16 16" fill="none" className="h-full w-full">
          <path
            d="M16 12V4C16 1.79086 14.2091 0 12 0H4"
            stroke="#7DD3FC"
            strokeWidth="3"
            strokeLinecap="round"
          />
        </svg>
      </div>

      <div className="pointer-events-none absolute inset-0 z-0 overflow-hidden rounded-[16px]">
        <svg
          viewBox="24 104 595 585"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
          className="h-full w-full"
          preserveAspectRatio="none"
        >
          <defs>
            <filter id="detailNeonGlow" x="-30%" y="-30%" width="160%" height="160%">
              <feGaussianBlur stdDeviation="3.5" result="blur" />
              <feMerge>
                <feMergeNode in="blur" />
                <feMergeNode in="SourceGraphic" />
              </feMerge>
            </filter>
          </defs>

          <g stroke="#94A3B8" strokeWidth="2.45" strokeLinecap="round" opacity="0.3">
            <path d="M640.302 181.184H435.126C396.47 181.184 377.142 200.604 377.142 239.443V306.665" />
            <path d="M640.302 593.475H479.73C441.073 593.475 421.745 612.895 421.745 651.734V709.993" />
            <path d="M20.3147 709.993V642.771C20.3147 615.882 33.6957 602.438 60.4577 602.438H180.887" />
            <path d="M640.302 181.184H638.319H586.778" strokeWidth="3.35" stroke="#64748B" opacity="0.45" />
          </g>

          <g style={{ color: '#0EA5E9' }} className="animate-neon">
            <path
              d="M640.302 181.184H435.126C396.47 181.184 377.142 200.604 377.142 239.443V306.665"
              stroke="currentColor"
              strokeWidth="2.45"
              strokeLinecap="round"
              pathLength="1"
              className="animate-flow"
              style={{ animationDuration: '3.8s' }}
            />
            <path
              d="M640.302 593.475H479.73C441.073 593.475 421.745 612.895 421.745 651.734V709.993"
              stroke="currentColor"
              strokeWidth="2.45"
              strokeLinecap="round"
              pathLength="1"
              className="animate-flow"
              style={{ animationDuration: '4.4s', animationDelay: '1.2s' }}
            />
            <path
              d="M20.3147 709.993V642.771C20.3147 615.882 33.6957 602.438 60.4577 602.438H180.887"
              stroke="currentColor"
              strokeWidth="2.45"
              strokeLinecap="round"
              pathLength="1"
              className="animate-flow"
              style={{ animationDuration: '4s', animationDelay: '0.6s' }}
            />

            <circle cx="377.142" cy="306.664" r="8" fill="currentColor" opacity="0.35" filter="url(#detailNeonGlow)" />
            <circle cx="377.142" cy="306.664" r="4.5" fill="currentColor" />
            <circle cx="377.142" cy="306.664" r="1.8" fill="#FFFFFF" opacity="0.9" />

            <circle cx="479.73" cy="593.475" r="8" fill="currentColor" opacity="0.35" filter="url(#detailNeonGlow)" />
            <circle cx="479.73" cy="593.475" r="4.5" fill="currentColor" />
            <circle cx="479.73" cy="593.475" r="1.8" fill="#FFFFFF" opacity="0.9" />

            <circle cx="180.887" cy="602.438" r="8" fill="currentColor" opacity="0.35" filter="url(#detailNeonGlow)" />
            <circle cx="180.887" cy="602.438" r="4.5" fill="currentColor" />
            <circle cx="180.887" cy="602.438" r="1.8" fill="#FFFFFF" opacity="0.9" />
          </g>
        </svg>
      </div>

      <div className="relative z-10 flex flex-col gap-5">
        <div className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-100 pb-3">
          <div className="flex items-center gap-2">
            <span className="font-semibold text-slate-900 text-sm">{ticket.id}</span>
            <span className="text-slate-300">·</span>
            {isInReview ? (
              <span className="inline-flex items-center gap-1.5 text-xs font-medium text-amber-600">
                <Clock className="h-3.5 w-3.5 text-amber-500" />
                <span>На рассмотрении</span>
              </span>
            ) : (
              <span className="inline-flex items-center gap-1.5 text-xs font-medium text-emerald-600">
                <CheckCircle2 className="h-3.5 w-3.5 text-emerald-500" />
                <span>Ответ получен</span>
              </span>
            )}
          </div>
          <time className="text-xs text-slate-400">
            {formatDate(ticket.createdAt)}
          </time>
        </div>

        <h1 className="text-xl font-bold leading-tight text-slate-900 sm:text-2xl">
          {ticket.subject}
        </h1>

        <div className="flex flex-col gap-2 rounded-xl border border-slate-200/90 bg-slate-50/60 p-2 sm:flex-row sm:items-center sm:justify-between sm:p-2.5">
          <div className="flex min-w-0 items-center gap-2 px-1">
            <LinkIcon className="h-4 w-4 shrink-0 text-sky-500" />
            <span className="truncate text-xs font-medium text-[#0284C7] sm:text-sm">
              {ticketUrl}
            </span>
          </div>

          <button
            type="button"
            onClick={handleCopyLink}
            className="inline-flex h-8 shrink-0 items-center justify-center gap-1.5 rounded-[8px] border border-slate-200 bg-white px-3 text-xs font-medium text-slate-700 shadow-2xs transition-colors hover:bg-slate-50 hover:text-slate-900 active:scale-[0.98]"
          >
            {copied ? (
              <>
                <Check className="h-3.5 w-3.5 text-emerald-600" />
                <span className="text-emerald-700 font-semibold">Скопировано!</span>
              </>
            ) : (
              <span>Копировать ссылку</span>
            )}
          </button>
        </div>

        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div className="flex flex-col gap-1.5 rounded-xl border border-slate-200/90 bg-slate-50/40 p-3 sm:p-3.5">
            <span className="text-[11px] font-semibold uppercase tracking-wider text-slate-400">
              ОТПРАВИТЕЛЬ
            </span>
            <div className="flex items-center gap-2.5">
              <div className="flex flex-col gap-0.5 text-slate-300">
                <span className="h-0.5 w-3.5 rounded-full bg-slate-300" />
                <span className="h-0.5 w-3.5 rounded-full bg-slate-300" />
                <span className="h-0.5 w-3.5 rounded-full bg-slate-300" />
              </div>
              <div className="flex min-w-0 flex-col">
                <span className="truncate font-bold text-slate-900 text-sm">
                  {ticket.author.name}
                </span>
                <span className="truncate text-xs text-slate-500">
                  {ticket.author.role}
                  {ticket.author.department ? ` · ${ticket.author.department}` : ''}
                </span>
              </div>
            </div>
          </div>

          <div className="flex flex-col gap-1.5 rounded-xl border border-slate-200/90 bg-slate-50/40 p-3 sm:p-3.5">
            <span className="text-[11px] font-semibold uppercase tracking-wider text-slate-400">
              ПОЛУЧАТЕЛЬ
            </span>
            <div className="flex items-center gap-2.5">
              <div className="flex flex-col gap-0.5 text-slate-300">
                <span className="h-0.5 w-3.5 rounded-full bg-slate-300" />
                <span className="h-0.5 w-3.5 rounded-full bg-slate-300" />
                <span className="h-0.5 w-3.5 rounded-full bg-slate-300" />
              </div>
              <div className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-sky-100 text-sky-600">
                <UserIcon className="h-3 w-3" />
              </div>
              <div className="flex min-w-0 flex-col">
                <span className="truncate font-bold text-slate-900 text-sm">
                  {ticket.recipient.name}
                </span>
                <span className="truncate text-xs text-slate-500">
                  {ticket.recipient.role}
                </span>
              </div>
            </div>
          </div>
        </div>

        <div className="flex flex-col gap-1.5">
          <span className="text-[11px] font-semibold uppercase tracking-wider text-slate-400">
            ОПИСАНИЕ ОБРАЩЕНИЯ
          </span>
          <div className="rounded-xl border border-slate-200/90 bg-white/95 p-4 text-xs leading-relaxed text-slate-700 sm:text-sm">
            {ticket.description}
          </div>
        </div>
      </div>
    </article>
  );
}

export default TicketDetailView;
