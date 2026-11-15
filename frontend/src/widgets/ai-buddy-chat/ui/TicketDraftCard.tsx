import { useEffect, memo } from 'react';
import { Check, Pencil, FileText, Loader2 } from 'lucide-react';
import { useShallow } from 'zustand/react/shallow';
import type { TicketDraftPayload } from '@/entities/chat';
import { useTicketDraftStore } from '../store/ticketDraftStore';

export interface TicketDraftCardProps {
  draft: TicketDraftPayload;
  onApprove?: (draft: TicketDraftPayload) => void | Promise<void>;
  onUpdate?: (updated: Partial<TicketDraftPayload>) => void;
  onCancel?: () => void;
  isSubmitting?: boolean;
}

export const TicketDraftCard = memo(function TicketDraftCard({
  draft,
  onApprove,
  onUpdate,
  isSubmitting = false,
}: TicketDraftCardProps) {
  const {
    isEditing,
    editedSubject,
    editedMessage,
    isStoreSubmitting,
  } = useTicketDraftStore(
    useShallow((state) => ({
      isEditing: state.isEditing,
      editedSubject: state.editedSubject,
      editedMessage: state.editedMessage,
      isStoreSubmitting: state.isSubmitting,
    }))
  );

  const startEditing = useTicketDraftStore((state) => state.startEditing);
  const cancelEditing = useTicketDraftStore((state) => state.cancelEditing);
  const setEditedSubject = useTicketDraftStore((state) => state.setEditedSubject);
  const setEditedMessage = useTicketDraftStore((state) => state.setEditedMessage);
  const saveEdit = useTicketDraftStore((state) => state.saveEdit);
  const approveDraft = useTicketDraftStore((state) => state.approveDraft);

  useEffect(() => {
    if (!isEditing) {
      setEditedSubject(draft.subject);
      setEditedMessage(draft.message);
    }
  }, [draft.subject, draft.message, isEditing, setEditedSubject, setEditedMessage]);

  const submitting = isSubmitting || isStoreSubmitting;

  const handleApprove = () => {
    void approveDraft(draft, onApprove);
  };

  const handleStartEdit = () => {
    startEditing(draft.subject, draft.message);
  };

  const handleCancelEdit = () => {
    cancelEditing(draft.subject, draft.message);
  };

  const handleSaveEdit = () => {
    saveEdit(onUpdate);
  };

  return (
    <div
      data-testid="ticket-draft-card"
      className="flex w-full flex-col gap-4 rounded-[20px] border border-[#BAE6FD] bg-[#F8FAFC] p-5 shadow-sm transition-all duration-200"
    >
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-200/70 pb-3.5">
        <div className="flex items-center gap-2 rounded-full bg-[#E0F2FE] px-3 py-1 text-xs font-semibold text-[#0284C7]">
          <FileText className="h-4 w-4 text-[#0284C7]" />
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
                onClick={handleStartEdit}
                className="flex items-center gap-1.5 rounded-xl border border-slate-300 bg-white px-3.5 py-2 text-xs font-bold text-[#334155] shadow-2xs transition-colors hover:bg-slate-100"
              >
                <Pencil className="h-3.5 w-3.5 text-[#334155]" />
                <span>Изменить</span>
              </button>

              <button
                type="button"
                onClick={handleApprove}
                disabled={submitting}
                className="flex items-center gap-2 rounded-xl bg-[#70C5FF] px-5 py-2 text-xs font-bold text-white shadow-[0_4px_12px_rgba(112,197,255,0.35)] transition-all duration-150 hover:bg-[#52B7FF] active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-50"
              >
                {submitting ? (
                  <Loader2 className="h-4 w-4 animate-spin text-white" />
                ) : (
                  <Check className="h-4 w-4 text-white" />
                )}
                <span>{submitting ? 'Отправка...' : 'Подтвердить и отправить'}</span>
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
});

export default TicketDraftCard;
