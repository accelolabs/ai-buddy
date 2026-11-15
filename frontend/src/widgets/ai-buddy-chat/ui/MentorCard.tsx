import type { MentorPayload } from '@/entities/chat';
import type { CreatedTicket } from '@/entities/chat/api/ticketApi';
import { useMentorCard } from '../model/useMentorCard';
import { MentorCardCompact } from './MentorCardCompact';
import { MentorCardForm } from './MentorCardForm';
import { MentorCardSuccess } from './MentorCardSuccess';

export interface MentorCardProps {
  mentor: MentorPayload;
  defaultOpen?: boolean;
  initialSubject?: string;
  initialMessage?: string;
  onContact?: (mentor: MentorPayload) => void;
  onTicketCreated?: (ticket: CreatedTicket) => void;
}

export function MentorCard({
  mentor,
  defaultOpen = false,
  initialSubject,
  initialMessage,
  onContact,
  onTicketCreated,
}: MentorCardProps) {
  const {
    viewState,
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
  } = useMentorCard({
    mentor,
    defaultOpen,
    initialSubject,
    initialMessage,
    onContact,
    onTicketCreated,
  });

  if (viewState === 'success' && createdTicket) {
    return <MentorCardSuccess ticket={createdTicket} />;
  }

  if (viewState === 'form') {
    return (
      <MentorCardForm
        mentor={mentor}
        subject={subject}
        message={message}
        errors={errors}
        isSubmitting={isSubmitting}
        onSubjectChange={handleSubjectChange}
        onMessageChange={handleMessageChange}
        onCancel={handleCancel}
        onSubmit={handleSubmit}
      />
    );
  }

  return <MentorCardCompact mentor={mentor} onOpenForm={handleOpenForm} />;
}

export default MentorCard;
