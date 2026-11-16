import { create } from 'zustand';
import type { MentorPayload } from '@/entities/chat';
import { createTicket, type CreatedTicket } from '@/entities/chat/api/ticketApi';
import { useChatStore } from '@/entities/chat';

export type MentorCardViewState = 'compact' | 'form' | 'success';

export interface MentorCardFormErrors {
  subject?: string;
  message?: string;
}

export interface MentorCardState {
  viewState: MentorCardViewState;
  subject: string;
  message: string;
  errors: MentorCardFormErrors;
  isSubmitting: boolean;
  createdTicket: CreatedTicket | null;
}

export interface MentorCardActions {
  setViewState: (viewState: MentorCardViewState) => void;
  setSubject: (subject: string) => void;
  setMessage: (message: string) => void;
  setErrors: (errors: MentorCardFormErrors) => void;
  openForm: (options?: { initialSubject?: string; initialMessage?: string; mentor?: MentorPayload }) => void;
  closeForm: () => void;
  reset: () => void;
  submitTicket: (
    mentorId: string,
    onSuccess?: (ticket: CreatedTicket) => void
  ) => Promise<boolean>;
}

export type MentorCardStore = MentorCardState & MentorCardActions;

export function resolveDefaultTicketData(customSubject?: string, customMessage?: string) {
  if (customSubject && customMessage) {
    return { subject: customSubject, message: customMessage };
  }

  const messages = useChatStore.getState().messages;
  const lastUserMsg = [...messages].reverse().find((m) => m.sender === 'user')?.text || '';

  const subject =
    customSubject ||
    (lastUserMsg
      ? `Вопрос по теме: ${lastUserMsg.length > 40 ? lastUserMsg.slice(0, 37) + '...' : lastUserMsg}`
      : 'Вопрос по материалам онбординга');

  const message =
    customMessage ||
    (lastUserMsg
      ? `Здравствуйте!\n\nНе удалось найти ответ в базе знаний по вопросу: «${lastUserMsg}».\n\nПрошу подсказать, где найти информацию или как лучше поступить.`
      : 'Здравствуйте! Прошу помочь разобраться с вопросом по стажировке.');

  return { subject, message };
}

export const useMentorCardStore = create<MentorCardStore>((set, get) => ({
  viewState: 'compact',
  subject: '',
  message: '',
  errors: {},
  isSubmitting: false,
  createdTicket: null,

  setViewState: (viewState) => set({ viewState }),

  setSubject: (subject) => {
    set((state) => ({
      subject,
      errors: state.errors.subject ? { ...state.errors, subject: undefined } : state.errors,
    }));
  },

  setMessage: (message) => {
    set((state) => ({
      message,
      errors: state.errors.message ? { ...state.errors, message: undefined } : state.errors,
    }));
  },

  setErrors: (errors) => set({ errors }),

  openForm: (options) => {
    const latest = resolveDefaultTicketData(options?.initialSubject, options?.initialMessage);
    const { subject, message } = get();

    set({
      viewState: 'form',
      subject: subject.trim() ? subject : latest.subject,
      message: message.trim() ? message : latest.message,
      errors: {},
    });
  },

  closeForm: () => {
    set({
      viewState: 'compact',
      errors: {},
    });
  },

  reset: () => {
    set({
      viewState: 'compact',
      subject: '',
      message: '',
      errors: {},
      isSubmitting: false,
      createdTicket: null,
    });
  },

  submitTicket: async (mentorId, onSuccess) => {
    const { subject, message, isSubmitting } = get();
    if (isSubmitting) return false;

    const newErrors: MentorCardFormErrors = {};
    if (!subject.trim()) {
      newErrors.subject = 'Укажите тему вопроса';
    }
    if (!message.trim()) {
      newErrors.message = 'Напишите текст сообщения';
    }

    if (Object.keys(newErrors).length > 0) {
      set({ errors: newErrors });
      return false;
    }

    set({ errors: {}, isSubmitting: true });

    try {
      const ticket = await createTicket({
        subject: subject.trim(),
        message: message.trim(),
        mentorId,
      });

      set({
        createdTicket: ticket,
        viewState: 'success',
        isSubmitting: false,
      });

      onSuccess?.(ticket);
      return true;
    } catch {
      set({
        errors: { message: 'Не удалось отправить тикет. Попробуйте еще раз.' },
        isSubmitting: false,
      });
      return false;
    }
  },
}));
