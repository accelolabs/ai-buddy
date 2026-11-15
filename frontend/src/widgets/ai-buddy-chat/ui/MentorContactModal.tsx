import { useState } from 'react';
import type { MentorPayload } from '@/entities/chat';
import { Dialog, DialogContent, DialogTitle, DialogDescription } from '@/shared/ui';
import { MentorContactForm, type MentorContactFormValues } from './MentorContactForm';

export interface MentorContactModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mentor?: MentorPayload | null;
  onSubmit?: (values: MentorContactFormValues) => void | Promise<void>;
}

export function MentorContactModal({
  open,
  onOpenChange,
  mentor,
  onSubmit,
}: MentorContactModalProps) {
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSubmit = async (values: MentorContactFormValues) => {
    try {
      setIsSubmitting(true);
      await onSubmit?.(values);
      onOpenChange(false);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-[540px] p-7 rounded-[24px]">
        <DialogTitle className="sr-only">Обращение к наставнику</DialogTitle>
        <DialogDescription className="sr-only">
          Форма отправки вопроса вашему наставнику
        </DialogDescription>

        <MentorContactForm
          mentor={mentor}
          isSubmitting={isSubmitting}
          onCancel={() => onOpenChange(false)}
          onSubmit={handleSubmit}
        />
      </DialogContent>
    </Dialog>
  );
}

export default MentorContactModal;
