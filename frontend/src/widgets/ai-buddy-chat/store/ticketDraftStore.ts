import { create } from 'zustand';
import type { TicketDraftPayload } from '@/entities/chat';

export interface TicketDraftState {
  isEditing: boolean;
  editedSubject: string;
  editedMessage: string;
  isSubmitting: boolean;
}

export interface TicketDraftActions {
  startEditing: (subject: string, message: string) => void;
  cancelEditing: (originalSubject: string, originalMessage: string) => void;
  setEditedSubject: (subject: string) => void;
  setEditedMessage: (message: string) => void;
  saveEdit: (onUpdate?: (updated: Partial<TicketDraftPayload>) => void) => boolean;
  approveDraft: (
    draft: TicketDraftPayload,
    onApprove?: (draft: TicketDraftPayload) => void | Promise<void>
  ) => Promise<void>;
  reset: () => void;
}

export type TicketDraftStore = TicketDraftState & TicketDraftActions;

export const useTicketDraftStore = create<TicketDraftStore>((set, get) => ({
  isEditing: false,
  editedSubject: '',
  editedMessage: '',
  isSubmitting: false,

  startEditing: (subject, message) => {
    set({
      isEditing: true,
      editedSubject: subject,
      editedMessage: message,
    });
  },

  cancelEditing: (originalSubject, originalMessage) => {
    set({
      isEditing: false,
      editedSubject: originalSubject,
      editedMessage: originalMessage,
    });
  },

  setEditedSubject: (editedSubject) => set({ editedSubject }),
  setEditedMessage: (editedMessage) => set({ editedMessage }),

  saveEdit: (onUpdate) => {
    const { editedSubject, editedMessage } = get();
    const trimmedSubject = editedSubject.trim();
    const trimmedMessage = editedMessage.trim();

    if (!trimmedSubject || !trimmedMessage) {
      return false;
    }

    onUpdate?.({
      subject: trimmedSubject,
      message: trimmedMessage,
    });

    set({ isEditing: false });
    return true;
  },

  approveDraft: async (draft, onApprove) => {
    const { isSubmitting } = get();
    if (isSubmitting) return;

    set({ isSubmitting: true });
    try {
      await onApprove?.(draft);
    } finally {
      set({ isSubmitting: false });
    }
  },

  reset: () => {
    set({
      isEditing: false,
      editedSubject: '',
      editedMessage: '',
      isSubmitting: false,
    });
  },
}));
