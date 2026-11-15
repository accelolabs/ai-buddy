import type { FormEvent } from 'react';
import { X, Clock, Send, Loader2 } from 'lucide-react';
import type { MentorPayload } from '@/entities/chat';
import type { MentorCardFormErrors } from '../store/mentorCardStore';

export interface MentorCardFormProps {
  mentor: MentorPayload;
  subject: string;
  message: string;
  errors: MentorCardFormErrors;
  isSubmitting: boolean;
  onSubjectChange: (value: string) => void;
  onMessageChange: (value: string) => void;
  onCancel: () => void;
  onSubmit: (e: FormEvent) => void;
}

export function MentorCardForm({
  mentor,
  subject,
  message,
  errors,
  isSubmitting,
  onSubjectChange,
  onMessageChange,
  onCancel,
  onSubmit,
}: MentorCardFormProps) {
  return (
    <form
      onSubmit={onSubmit}
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
          onClick={onCancel}
          disabled={isSubmitting}
          className="flex h-8 w-8 items-center justify-center rounded-lg text-[#94A3B8] transition-colors hover:bg-slate-100 hover:text-[#475569] disabled:opacity-50"
          title="Свернуть форму"
          aria-label="Свернуть форму"
        >
          <X className="h-5 w-5" />
        </button>
      </div>

      <div className="flex items-start gap-2.5 rounded-[12px] border border-[#E0F2FE] bg-[#F4FAFF] p-3 text-[13px] leading-relaxed text-[#334155]">
        <Clock className="mt-0.5 h-4 w-4 shrink-0 text-[#0284C7]" />
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
          onChange={(e) => onSubjectChange(e.target.value)}
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
          onChange={(e) => onMessageChange(e.target.value)}
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
          onClick={onCancel}
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
          {isSubmitting ? (
            <Loader2 className="h-4 w-4 animate-spin text-white" />
          ) : (
            <Send className="h-4 w-4 shrink-0 text-white" />
          )}
          <span>{isSubmitting ? 'Отправка...' : 'Отправить наставнику'}</span>
        </button>
      </div>
    </form>
  );
}

export default MentorCardForm;
