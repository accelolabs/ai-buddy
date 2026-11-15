import { useEffect, type FormEvent } from 'react';
import { useShallow } from 'zustand/react/shallow';
import type { MentorPayload } from '@/entities/chat';
import type { CreatedTicket } from '@/entities/chat/api/ticketApi';
import {
  useMentorCardStore,
  resolveDefaultTicketData,
  type MentorCardViewState,
  type MentorCardFormErrors,
} from '../store/mentorCardStore';

export type { MentorCardViewState, MentorCardFormErrors };

export interface UseMentorCardOptions {
  mentor: MentorPayload;
  defaultOpen?: boolean;
  initialSubject?: string;
  initialMessage?: string;
  onContact?: (mentor: MentorPayload) => void;
  onTicketCreated?: (ticket: CreatedTicket) => void;
}

export function useMentorCard({
  mentor,
  defaultOpen = false,
  initialSubject,
  initialMessage,
  onContact,
  onTicketCreated,
}: UseMentorCardOptions) {
  const {
    viewState,
    subject,
    message,
    errors,
    isSubmitting,
    createdTicket,
  } = useMentorCardStore(
    useShallow((state) => ({
      viewState: state.viewState,
      subject: state.subject,
      message: state.message,
      errors: state.errors,
      isSubmitting: state.isSubmitting,
      createdTicket: state.createdTicket,
    }))
  );

  const setViewState = useMentorCardStore((state) => state.setViewState);
  const setSubject = useMentorCardStore((state) => state.setSubject);
  const setMessage = useMentorCardStore((state) => state.setMessage);
  const openForm = useMentorCardStore((state) => state.openForm);
  const closeForm = useMentorCardStore((state) => state.closeForm);
  const submitTicket = useMentorCardStore((state) => state.submitTicket);

  useEffect(() => {
    if (defaultOpen && viewState === 'compact') {
      const defaults = resolveDefaultTicketData(initialSubject, initialMessage);
      setSubject(defaults.subject);
      setMessage(defaults.message);
      setViewState('form');
    }
  }, [defaultOpen, viewState, initialSubject, initialMessage, setSubject, setMessage, setViewState]);

  const handleOpenForm = () => {
    openForm({ initialSubject, initialMessage, mentor });
    onContact?.(mentor);
  };

  const handleCancel = () => {
    closeForm();
  };

  const handleSubjectChange = (val: string) => {
    setSubject(val);
  };

  const handleMessageChange = (val: string) => {
    setMessage(val);
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    await submitTicket(mentor.id, onTicketCreated);
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
