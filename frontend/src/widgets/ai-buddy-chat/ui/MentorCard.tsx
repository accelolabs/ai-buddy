import { useState, type FormEvent } from 'react';
import type { MentorPayload } from '@/entities/chat';
import { createTicket, type CreatedTicket } from '@/entities/chat/api/ticketApi';

export interface MentorCardProps {
  mentor: MentorPayload;
  defaultOpen?: boolean;
  onContact?: (mentor: MentorPayload) => void;
  onTicketCreated?: (ticket: CreatedTicket) => void;
}

const MAIL_ICON = (
  <svg
    className="h-4 w-4 text-white"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <path d="M4 4h16c1.1 0 2 .9 2 2v12c0 1.1-.9 2-2 2H4c-1.1 0-2-.9-2-2V6c0-1.1.9-2 2-2z" />
    <polyline points="22,6 12,13 2,6" />
  </svg>
);

const CLOSE_ICON = (
  <svg
    className="h-5 w-5"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <path d="M18 6 6 18" />
    <path d="m6 6 12 12" />
  </svg>
);

const CLOCK_ICON = (
  <svg
    className="h-4 w-4 shrink-0 text-[#0284C7]"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <circle cx="12" cy="12" r="10" />
    <polyline points="12 6 12 12 16 14" />
  </svg>
);

const SEND_PLANE_ICON = (
  <svg
    className="h-4 w-4 shrink-0 text-white"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <path d="m22 2-7 20-4-9-9-4Z" />
    <path d="M22 2 11 13" />
  </svg>
);

const CHECK_ICON = (
  <svg
    className="h-4 w-4 text-emerald-600"
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

const EXTERNAL_LINK_ICON = (
  <svg
    className="h-3.5 w-3.5 inline-block shrink-0"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2.2"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" />
    <polyline points="15 3 21 3 21 9" />
    <line x1="10" y1="14" x2="21" y2="3" />
  </svg>
);

export function MentorCard({
  mentor,
  defaultOpen = false,
  onContact,
  onTicketCreated,
}: MentorCardProps) {
  const [viewState, setViewState] = useState<'compact' | 'form' | 'success'>(
    defaultOpen ? 'form' : 'compact',
  );
  const [subject, setSubject] = useState('');
  const [message, setMessage] = useState('');
  const [errors, setErrors] = useState<{ subject?: string; message?: string }>({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [createdTicket, setCreatedTicket] = useState<CreatedTicket | null>(null);

  const handleOpenForm = () => {
    setViewState('form');
    onContact?.(mentor);
  };

  const handleCancel = () => {
    setViewState('compact');
    setErrors({});
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();

    const newErrors: { subject?: string; message?: string } = {};
    if (!subject.trim()) {
      newErrors.subject = 'Укажите тему вопроса';
    }
    if (!message.trim()) {
      newErrors.message = 'Напишите текст сообщения';
    }

    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors);
      return;
    }

    setErrors({});
    setIsSubmitting(true);

    try {
      const ticket = await createTicket({
        subject: subject.trim(),
        message: message.trim(),
        mentorId: mentor.id,
      });

      setCreatedTicket(ticket);
      setViewState('success');
      onTicketCreated?.(ticket);
    } catch {
      setErrors({ message: 'Не удалось отправить тикет. Попробуйте еще раз.' });
    } finally {
      setIsSubmitting(false);
    }
  };

  if (viewState === 'success' && createdTicket) {
    return (
      <div
        data-testid="mentor-ticket-success"
        className="flex w-full flex-wrap items-center justify-between gap-3 rounded-[16px] border border-emerald-200 bg-emerald-50/70 p-4 text-[14px] leading-snug text-[#0F172A] shadow-xs transition-all duration-300 animate-in fade-in zoom-in-95"
      >
        <div className="flex items-center gap-2.5">
          <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-emerald-100 ring-2 ring-emerald-200/50">
            {CHECK_ICON}
          </div>
          <div>
            Успешно отправлен тикет{' '}
            <strong className="font-semibold text-emerald-950">«{createdTicket.subject}»</strong>.
          </div>
        </div>

        <a
          href={createdTicket.url}
          target="_blank"
          rel="noopener noreferrer"
          title={`Перейти к тикету #${createdTicket.id}`}
          className="inline-flex items-center gap-1.5 rounded-xl bg-white px-3.5 py-1.5 font-bold text-[#0284C7] shadow-xs ring-1 ring-slate-200 transition-all duration-150 hover:bg-[#F0F9FF] hover:text-[#0369A1] hover:ring-[#BAE6FD] active:scale-[0.98]"
        >
          <span>Перейти к тикету #{createdTicket.id}</span>
          {EXTERNAL_LINK_ICON}
        </a>
      </div>
    );
  }

  if (viewState === 'form') {
    return (
      <form
        onSubmit={handleSubmit}
        noValidate
        data-testid="mentor-contact-form-inline"
        className="flex w-full flex-col gap-4 rounded-[20px] border border-[#BAE6FD] bg-white p-5 shadow-sm transition-all duration-300 animate-in fade-in zoom-in-95"
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="relative flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-[#70B9FA] text-sm font-bold text-[#0F172A]">
              {mentor.initials}
              {mentor.isOnline && (
                <span
                  className="absolute bottom-0 right-0 h-3 w-3 rounded-full border-2 border-white bg-[#00C853]"
                  title="В сети"
                />
              )}
            </div>

            <div className="flex flex-col">
              <span className="text-base font-bold text-[#0F172A] leading-tight">
                {mentor.name}
              </span>
              <span className="text-xs text-[#64748B]">
                {mentor.role} · {mentor.team}
              </span>
            </div>
          </div>

          <button
            type="button"
            onClick={handleCancel}
            disabled={isSubmitting}
            className="flex h-8 w-8 items-center justify-center rounded-lg text-[#94A3B8] transition-colors hover:bg-slate-100 hover:text-[#475569] disabled:opacity-50"
            title="Свернуть форму"
            aria-label="Свернуть форму"
          >
            {CLOSE_ICON}
          </button>
        </div>

        <div className="flex items-start gap-2.5 rounded-[12px] border border-[#E0F2FE] bg-[#F4FAFF] p-3 text-[13px] leading-relaxed text-[#334155]">
          <div className="mt-0.5">{CLOCK_ICON}</div>
          <div className="flex flex-col">
            <div>
              <strong className="font-semibold text-[#0F172A]">График наставника:</strong>{' '}
              <span>{mentor.schedule || 'Отвечает в будни с 10:00 до 19:00.'}</span>
            </div>
            <div>Среднее время первого ответа — 15 минут.</div>
          </div>
        </div>

        <div className="flex flex-col gap-1.5">
          <label
            htmlFor={`mentor-subject-${mentor.id}`}
            className="text-xs font-bold text-[#0F172A]"
          >
            Тема вопроса
          </label>
          <input
            id={`mentor-subject-${mentor.id}`}
            type="text"
            value={subject}
            onChange={(e) => {
              setSubject(e.target.value);
              if (errors.subject) setErrors((prev) => ({ ...prev, subject: undefined }));
            }}
            disabled={isSubmitting}
            placeholder="Коротко сформулируйте тему..."
            className={`h-10 w-full rounded-xl border bg-[#F8FAFC] px-3.5 text-sm text-[#0F172A] transition-all outline-none placeholder:text-[#94A3B8] focus:bg-white focus:border-[#38BDF8] focus:ring-2 focus:ring-[#38BDF8]/20 ${
              errors.subject ? 'border-red-500 bg-red-50/30' : 'border-[#E2E8F0]'
            }`}
          />
          {errors.subject && <span className="text-xs text-red-500">{errors.subject}</span>}
        </div>

        <div className="flex flex-col gap-1.5">
          <label
            htmlFor={`mentor-message-${mentor.id}`}
            className="text-xs font-bold text-[#0F172A]"
          >
            Ваше сообщение
          </label>
          <textarea
            id={`mentor-message-${mentor.id}`}
            rows={3}
            value={message}
            onChange={(e) => {
              setMessage(e.target.value);
              if (errors.message) setErrors((prev) => ({ ...prev, message: undefined }));
            }}
            disabled={isSubmitting}
            placeholder="Опишите, на каком этапе возникла сложность или укажите номер задачи..."
            className={`w-full resize-none rounded-xl border bg-white p-3 text-sm text-[#0F172A] transition-all outline-none placeholder:text-[#94A3B8] focus:ring-2 focus:ring-[#38BDF8]/20 ${
              errors.message
                ? 'border-red-500 bg-red-50/30'
                : 'border-[#38BDF8] focus:border-[#0284C7]'
            }`}
          />
          {errors.message && <span className="text-xs text-red-500">{errors.message}</span>}
        </div>

        <div className="flex items-center justify-end gap-2.5 pt-1">
          <button
            type="button"
            onClick={handleCancel}
            disabled={isSubmitting}
            className="rounded-xl px-3.5 py-2 text-xs font-bold text-[#334155] transition-colors hover:bg-slate-100 hover:text-[#0F172A] disabled:opacity-50"
          >
            Отмена
          </button>

          <button
            type="submit"
            disabled={isSubmitting}
            className="flex items-center gap-2 rounded-xl bg-[#00AAFF] px-5 py-2 text-xs font-bold text-white shadow-xs transition-all duration-200 hover:bg-[#0096E6] active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-50"
          >
            {SEND_PLANE_ICON}
            <span>{isSubmitting ? 'Отправка...' : 'Отправить наставнику'}</span>
          </button>
        </div>
      </form>
    );
  }

  return (
    <div
      data-testid="mentor-card-compact"
      className="flex w-full flex-wrap items-center justify-between gap-4 rounded-[16px] border border-slate-200 bg-white p-4 shadow-xs transition-all duration-200"
    >
      <div className="flex items-center gap-3.5">
        <div className="relative flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-[#89C6FC] text-sm font-bold text-[#004B87]">
          {mentor.initials}
          {mentor.isOnline && (
            <span
              className="absolute -bottom-0.5 -right-0.5 h-3.5 w-3.5 rounded-full border-2 border-white bg-[#04E061]"
              title="В сети"
            />
          )}
        </div>

        <div className="flex flex-col gap-0.5">
          <span className="text-sm font-bold text-[#0F172A]">{mentor.name}</span>
          <span className="text-xs text-[#64748B]">
            {mentor.role} · {mentor.team}
          </span>
          <span className="text-xs text-[#64748B]">{mentor.schedule}</span>
        </div>
      </div>

      <button
        type="button"
        onClick={handleOpenForm}
        className="flex items-center gap-2 rounded-xl bg-[#00AAFF] px-4 py-2 text-sm font-medium text-white transition-all duration-200 hover:bg-[#0096E6] hover:shadow-xs active:scale-[0.98]"
      >
        {MAIL_ICON}
        <span>Сообщить</span>
      </button>
    </div>
  );
}

export default MentorCard;
