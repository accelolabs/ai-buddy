import { useState, type FormEvent } from 'react';
import type { MentorPayload } from '@/entities/chat';

export interface MentorContactFormValues {
  subject: string;
  message: string;
  mentorId: string;
}

export interface MentorContactFormProps {
  mentor?: MentorPayload | null;
  initialSubject?: string;
  initialMessage?: string;
  isSubmitting?: boolean;
  onCancel?: () => void;
  onSubmit?: (values: MentorContactFormValues) => void | Promise<void>;
}

const DEFAULT_MENTOR: MentorPayload = {
  id: 'mentor-1',
  name: 'Мария Соколова',
  role: 'Наставник стажёров',
  team: 'Команда онбординга',
  schedule: 'Отвечает в будни с 10:00 до 19:00',
  initials: 'МС',
  isOnline: true,
};

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

export function MentorContactForm({
  mentor = DEFAULT_MENTOR,
  initialSubject = '',
  initialMessage = '',
  isSubmitting = false,
  onCancel,
  onSubmit,
}: MentorContactFormProps) {
  const currentMentor = mentor || DEFAULT_MENTOR;

  const [subject, setSubject] = useState(initialSubject);
  const [message, setMessage] = useState(initialMessage);
  const [errors, setErrors] = useState<{ subject?: string; message?: string }>({});

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
    await onSubmit?.({
      subject: subject.trim(),
      message: message.trim(),
      mentorId: currentMentor.id,
    });
  };

  return (
    <form
      onSubmit={handleSubmit}
      className="flex w-full flex-col gap-5 text-[#0F172A]"
      noValidate
      data-testid="mentor-contact-form"
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3.5">
          <div className="relative flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-[#70B9FA] text-base font-bold text-[#0F172A]">
            {currentMentor.initials}
            {currentMentor.isOnline && (
              <span
                className="absolute bottom-0 right-0 h-3.5 w-3.5 rounded-full border-2 border-white bg-[#00C853]"
                title="В сети"
                aria-label="В сети"
              />
            )}
          </div>

          <div className="flex flex-col">
            <span className="text-lg font-bold text-[#0F172A] leading-tight">
              {currentMentor.name}
            </span>
            <span className="text-sm text-[#64748B]">
              {currentMentor.role} · {currentMentor.team}
            </span>
          </div>
        </div>

        {onCancel && (
          <button
            type="button"
            onClick={onCancel}
            className="flex h-8 w-8 items-center justify-center rounded-lg text-[#94A3B8] transition-colors hover:bg-slate-100 hover:text-[#475569]"
            title="Закрыть"
            aria-label="Закрыть"
          >
            {CLOSE_ICON}
          </button>
        )}
      </div>

      <div className="flex items-start gap-3 rounded-[14px] border border-[#E0F2FE] bg-[#F4FAFF] p-3.5 text-[13px] leading-relaxed text-[#334155]">
        <div className="mt-0.5">{CLOCK_ICON}</div>
        <div className="flex flex-col">
          <div>
            <strong className="font-bold text-[#0F172A]">График наставника:</strong>{' '}
            <span>{currentMentor.schedule || 'Отвечает в будни с 10:00 до 19:00.'}</span>
          </div>
          <div>Среднее время первого ответа — 15 минут.</div>
        </div>
      </div>

      <div className="flex flex-col gap-1.5">
        <label htmlFor="mentor-subject" className="text-sm font-bold text-[#0F172A]">
          Тема вопроса
        </label>
        <input
          id="mentor-subject"
          type="text"
          value={subject}
          onChange={(e) => {
            setSubject(e.target.value);
            if (errors.subject) setErrors((prev) => ({ ...prev, subject: undefined }));
          }}
          disabled={isSubmitting}
          className={`h-11 w-full rounded-xl border bg-[#F8FAFC] px-3.5 text-sm text-[#0F172A] transition-all outline-none placeholder:text-[#94A3B8] focus:bg-white focus:border-[#38BDF8] focus:ring-2 focus:ring-[#38BDF8]/20 ${
            errors.subject ? 'border-red-500 bg-red-50/30' : 'border-[#E2E8F0]'
          }`}
        />
        {errors.subject && <span className="text-xs text-red-500">{errors.subject}</span>}
      </div>

      <div className="flex flex-col gap-1.5">
        <label htmlFor="mentor-message" className="text-sm font-bold text-[#0F172A]">
          Ваше сообщение
        </label>
        <textarea
          id="mentor-message"
          rows={4}
          value={message}
          onChange={(e) => {
            setMessage(e.target.value);
            if (errors.message) setErrors((prev) => ({ ...prev, message: undefined }));
          }}
          disabled={isSubmitting}
          placeholder="Опишите, на каком этапе возникла сложность или укажите номер тикета..."
          className={`w-full resize-none rounded-2xl border bg-white p-3.5 text-sm text-[#0F172A] transition-all outline-none placeholder:text-[#94A3B8] focus:ring-2 focus:ring-[#38BDF8]/20 ${
            errors.message
              ? 'border-red-500 bg-red-50/30'
              : 'border-[#38BDF8] focus:border-[#0284C7]'
          }`}
        />
        {errors.message && <span className="text-xs text-red-500">{errors.message}</span>}
      </div>

      <div className="mt-2 flex items-center justify-end gap-3">
        {onCancel && (
          <button
            type="button"
            onClick={onCancel}
            disabled={isSubmitting}
            className="rounded-xl px-4 py-2.5 text-sm font-bold text-[#334155] transition-colors hover:bg-slate-100 hover:text-[#0F172A] disabled:opacity-50"
          >
            Отмена
          </button>
        )}

        <button
          type="submit"
          disabled={isSubmitting}
          className="flex items-center gap-2 rounded-[14px] bg-[#70C5FF] px-6 py-2.5 text-sm font-bold text-white shadow-[0_4px_12px_rgba(112,197,255,0.35)] transition-all duration-200 hover:bg-[#52B7FF] active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-50"
        >
          {SEND_PLANE_ICON}
          <span>{isSubmitting ? 'Отправка...' : 'Отправить наставнику'}</span>
        </button>
      </div>
    </form>
  );
}

export default MentorContactForm;
