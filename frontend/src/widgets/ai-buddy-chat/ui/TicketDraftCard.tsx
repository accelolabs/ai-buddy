import { useState } from 'react';
import type { TicketDraftPayload } from '@/entities/chat';

export interface TicketDraftCardProps {
  draft: TicketDraftPayload;
  onApprove?: (draft: TicketDraftPayload) => void | Promise<void>;
  onUpdate?: (updated: Partial<TicketDraftPayload>) => void;
  onCancel?: () => void;
  isSubmitting?: boolean;
}

const CHECK_ICON = (
  <svg
    className="h-4 w-4 shrink-0 text-white"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2.5"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <polyline points="20 6 9 17 4 12" />
  </svg>
);

const PENCIL_ICON = (
  <svg
    className="h-3.5 w-3.5 shrink-0"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z" />
    <path d="m15 5 4 4" />
  </svg>
);

const FILE_TEXT_ICON = (
  <svg
    className="h-4 w-4 text-[#0284C7]"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <path d="M14.5 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7.5L14.5 2z" />
    <polyline points="14 2 14 8 20 8" />
    <line x1="16" y1="13" x2="8" y2="13" />
    <line x1="16" y1="17" x2="8" y2="17" />
    <line x1="10" y1="9" x2="8" y2="9" />
  </svg>
);

export function TicketDraftCard({
  draft,
  onApprove,
  onUpdate,
  isSubmitting = false,
}: TicketDraftCardProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [localSubmitting, setLocalSubmitting] = useState(false);
  const [editedSubject, setEditedSubject] = useState(draft.subject);
  const [editedMessage, setEditedMessage] = useState(draft.message);

  const submitting = isSubmitting || localSubmitting;

  const handleApprove = async () => {
    if (submitting) return;
    setLocalSubmitting(true);
    try {
      await onApprove?.(draft);
    } finally {
      setLocalSubmitting(false);
    }
  };

  const handleSaveEdit = () => {
    if (!editedSubject.trim() || !editedMessage.trim()) return;
    onUpdate?.({
      subject: editedSubject.trim(),
      message: editedMessage.trim(),
    });
    setIsEditing(false);
  };

  const handleCancelEdit = () => {
    setEditedSubject(draft.subject);
    setEditedMessage(draft.message);
    setIsEditing(false);
  };

  return (
    <div
      data-testid="ticket-draft-card"
      className="flex w-full flex-col gap-4 rounded-[20px] border border-[#BAE6FD] bg-[#F8FAFC] p-5 shadow-sm transition-all duration-200"
    >
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-200/70 pb-3.5">
        <div className="flex items-center gap-2 rounded-full bg-[#E0F2FE] px-3 py-1 text-xs font-semibold text-[#0284C7]">
          {FILE_TEXT_ICON}
          <span>Черновик тикета</span>
        </div>

        <div className="flex items-center gap-2.5">
          <div className="relative flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-[#70B9FA] text-xs font-bold text-[#0F172A]">
            {draft.mentor.initials}
            {draft.mentor.isOnline && (
              <span
                className="absolute -bottom-0.5 -right-0.5 h-2.5 w-2.5 rounded-full border-2 border-white bg-[#00C853]"
                title="В сети"
              />
            )}
          </div>
          <span className="text-xs text-[#64748B]">
            Получатель:{' '}
            <strong className="font-semibold text-[#0F172A]">{draft.mentor.name}</strong>
          </span>
        </div>
      </div>

      <div className="flex flex-col gap-1.5">
        <label className="text-xs font-bold tracking-wide text-[#475569] uppercase">
          Тема обращения
        </label>

        {isEditing ? (
          <input
            type="text"
            value={editedSubject}
            onChange={(e) => setEditedSubject(e.target.value)}
            className="h-10 w-full rounded-xl border border-[#38BDF8] bg-white px-3.5 text-sm font-semibold text-[#0F172A] shadow-2xs outline-none focus:ring-2 focus:ring-[#38BDF8]/20"
            placeholder="Укажите тему обращения..."
          />
        ) : (
          <div className="rounded-xl border border-slate-200/80 bg-white px-3.5 py-2.5 text-sm font-bold text-[#0F172A] shadow-2xs">
            {draft.subject}
          </div>
        )}
      </div>

      <div className="flex flex-col gap-1.5">
        <label className="text-xs font-bold tracking-wide text-[#475569] uppercase">
          Текст сообщения
        </label>
        {isEditing ? (
          <textarea
            rows={4}
            value={editedMessage}
            onChange={(e) => setEditedMessage(e.target.value)}
            className="w-full resize-none rounded-xl border border-[#38BDF8] bg-white p-3.5 text-sm leading-relaxed text-[#0F172A] shadow-2xs outline-none focus:ring-2 focus:ring-[#38BDF8]/20"
            placeholder="Опишите проблему подробно..."
          />
        ) : (
          <div className="whitespace-pre-wrap rounded-xl border border-slate-200/80 bg-white p-3.5 text-[14px] leading-relaxed text-[#334155] shadow-2xs">
            {draft.message}
          </div>
        )}
      </div>

      <div className="flex flex-col gap-2 pt-1">
        {isEditing ? (
          <div className="flex items-center justify-end gap-2.5">
            <button
              type="button"
              onClick={handleCancelEdit}
              className="rounded-xl px-4 py-2 text-xs font-bold text-[#64748B] transition-colors hover:bg-slate-200/60 hover:text-[#0F172A]"
            >
              Отмена
            </button>
            <button
              type="button"
              onClick={handleSaveEdit}
              className="rounded-xl bg-[#0284C7] px-4 py-2 text-xs font-bold text-white shadow-xs transition-colors hover:bg-[#0369A1]"
            >
              Сохранить изменения
            </button>
          </div>
        ) : (
          <div className="flex flex-wrap items-center justify-between gap-3">
            <span className="text-xs text-[#64748B]">
              Или напишите в чат, например: <em>«измени тему на...»</em>
            </span>

            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => setIsEditing(true)}
                className="flex items-center gap-1.5 rounded-xl border border-slate-300 bg-white px-3.5 py-2 text-xs font-bold text-[#334155] shadow-2xs transition-colors hover:bg-slate-100"
              >
                {PENCIL_ICON}
                <span>Изменить</span>
              </button>

              <button
                type="button"
                onClick={handleApprove}
                disabled={submitting}
                className="flex items-center gap-2 rounded-xl bg-[#70C5FF] px-5 py-2 text-xs font-bold text-white shadow-[0_4px_12px_rgba(112,197,255,0.35)] transition-all duration-150 hover:bg-[#52B7FF] active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-50"
              >
                {CHECK_ICON}
                <span>{submitting ? 'Отправка...' : 'Подтвердить и отправить'}</span>
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default TicketDraftCard;
