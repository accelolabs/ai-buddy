import { useState, type FormEvent } from 'react';
import type { MentorPayload } from '@/entities/chat';
import { createTicket, type CreatedTicket } from '@/entities/chat/api/ticketApi';
import { useChatStore } from '@/entities/chat';

export type MentorCardViewState = 'compact' | 'form' | 'success';

export interface MentorCardFormErrors {
  subject?: string;
  message?: string;
}

export interface UseMentorCardOptions {
  mentor: MentorPayload;
  defaultOpen?: boolean;
  initialSubject?: string;
  initialMessage?: string;
  onContact?: (mentor: MentorPayload) => void;
  onTicketCreated?: (ticket: CreatedTicket) => void;
}

function getDefaultTicketData(customSubject?: string, customMessage?: string) {
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

export function useMentorCard({
  mentor,
  defaultOpen = false,
  initialSubject,
  initialMessage,
  onContact,
  onTicketCreated,
}: UseMentorCardOptions) {
  const defaults = getDefaultTicketData(initialSubject, initialMessage);

  const [viewState, setViewState] = useState<MentorCardViewState>(
    defaultOpen ? 'form' : 'compact',
  );
  const [subject, setSubject] = useState(defaults.subject);
  const [message, setMessage] = useState(defaults.message);
  const [errors, setErrors] = useState<MentorCardFormErrors>({});
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [createdTicket, setCreatedTicket] = useState<CreatedTicket | null>(null);

  const handleOpenForm = () => {
    const latest = getDefaultTicketData(initialSubject, initialMessage);
    if (!subject.trim()) {
      setSubject(latest.subject);
    }
    if (!message.trim()) {
      setMessage(latest.message);
    }
    setViewState('form');
    onContact?.(mentor);
  };

  const handleCancel = () => {
    setViewState('compact');
    setErrors({});
  };

  const handleSubjectChange = (val: string) => {
    setSubject(val);
    if (errors.subject) {
      setErrors((prev) => ({ ...prev, subject: undefined }));
    }
  };

  const handleMessageChange = (val: string) => {
    setMessage(val);
    if (errors.message) {
      setErrors((prev) => ({ ...prev, message: undefined }));
    }
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();

    const newErrors: MentorCardFormErrors = {};
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

  return {
    viewState,
    setViewState,
    subject,
    message,
    errors,
    isSubmitting,
    createdTicket,
    handleOpenForm,
    handleCancel,
    handleSubjectChange,
    handleMessageChange,
    handleSubmit,
  };
}
