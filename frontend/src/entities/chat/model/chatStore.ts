import { create } from 'zustand';
import type { ChatMessage, SSEEvent, TicketDraftPayload, TicketFlowStep } from './types';
import { simulateSSEStream } from '../api/chatStream';
import { createTicket } from '../api/ticketApi';
import { createUserMessage, createBuddyMessage, updateMessagesWithEvent } from './helpers';
import { queryClient } from '@/shared/api';

export interface ChatState {
  draft: string;
  isExpanded: boolean;
  messages: ChatMessage[];
  isStreaming: boolean;
  ticketFlowStep: TicketFlowStep;
  activeTicketDraft: TicketDraftPayload | null;
  activeDialogId: string | null;
}

export interface ChatActions {
  setDraft: (text: string) => void;
  setIsExpanded: (expanded: boolean) => void;
  applyPrompt: (promptText: string) => void;
  resetDraft: () => void;
  setActiveDialogId: (id: string | null) => void;
  startNewChat: () => void;
  sendMessage: (text: string) => Promise<void>;
  retryMessage: (buddyMessageId?: string) => Promise<void>;
  applySSEEvent: (buddyMessageId: string, event: SSEEvent) => void;
  sendMentorTicket: (payload: { subject: string; message: string; mentorId: string }) => void;
  approveTicketDraft: (draft?: TicketDraftPayload) => Promise<void>;
  updateTicketDraft: (updated: Partial<TicketDraftPayload>) => void;
  cancelTicketDraft: () => void;
  clearMessages: () => void;
  stopStreaming: () => void;
}

export type ChatStore = ChatState & ChatActions;

const initialChatState: ChatState = {
  draft: '',
  isExpanded: false,
  messages: [],
  isStreaming: false,
  ticketFlowStep: 'idle',
  activeTicketDraft: null,
  activeDialogId: null,
};

let activeAbortController: AbortController | null = null;

function abortActiveStream() {
  if (activeAbortController) {
    activeAbortController.abort();
    activeAbortController = null;
  }
}

export const useChatStore = create<ChatStore>((set, get) => ({
  ...initialChatState,

  setDraft: (draft) => set({ draft }),
  setIsExpanded: (isExpanded) => set({ isExpanded }),
  resetDraft: () => set({ draft: '' }),
  setActiveDialogId: (activeDialogId) => set({ activeDialogId }),

  startNewChat: () => {
    abortActiveStream();
    set({
      activeDialogId: null,
      messages: [],
      draft: '',
      isStreaming: false,
      ticketFlowStep: 'idle',
      activeTicketDraft: null,
    });
  },

  applyPrompt: (promptText) => {
    set({ draft: promptText, isExpanded: true });
    get().sendMessage(promptText);
  },

  applySSEEvent: (buddyMessageId, event) => {
    set((state) => {
      let nextFlowStep = state.ticketFlowStep;
      let nextActiveDraft = state.activeTicketDraft;

      if (event.type === 'flow_step') {
        nextFlowStep = event.step;
        if (event.draft !== undefined) {
          nextActiveDraft = event.draft;
        } else if (event.step === 'idle') {
          nextActiveDraft = null;
        }
      }

      return {
        messages: updateMessagesWithEvent(state.messages, buddyMessageId, event),
        isStreaming: event.type === 'done' || event.type === 'error' ? false : state.isStreaming,
        ticketFlowStep: nextFlowStep,
        activeTicketDraft: nextActiveDraft,
      };
    });
  },

  sendMentorTicket: (payload) => {
    const confirmationMsg: ChatMessage = {
      id: `buddy-${Date.now()}`,
      sender: 'buddy',
      text: `Ваше обращение на тему «${payload.subject}» отправлено наставнику. Среднее время первого ответа — 15 минут.`,
      status: 'completed',
      createdAt: new Date().toISOString(),
    };
    set((state) => ({
      messages: [...state.messages, confirmationMsg],
      ticketFlowStep: 'idle',
      activeTicketDraft: null,
    }));
  },

  approveTicketDraft: async (draft) => {
    const targetDraft = draft || get().activeTicketDraft;
    if (!targetDraft) return;

    try {
      const ticket = await createTicket({
        subject: targetDraft.subject,
        message: targetDraft.message,
        mentorId: targetDraft.mentor.id,
      });

      set((state) => {
        const updatedMessages = state.messages.map((msg) => {
          if (!msg.cards) return msg;
          const nextCards = msg.cards.map((card) => {
            if (card.type === 'ticket_draft') {
              return {
                type: 'ticket_success' as const,
                payload: ticket,
              };
            }
            return card;
          });
          return { ...msg, cards: nextCards };
        });

        return {
          messages: updatedMessages,
          ticketFlowStep: 'idle',
          activeTicketDraft: null,
        };
      });
    } catch (err) {
      void err;
    }
  },

  updateTicketDraft: (updated) => {
    set((state) => {
      if (!state.activeTicketDraft) return state;
      const nextDraft: TicketDraftPayload = {
        ...state.activeTicketDraft,
        ...updated,
      };

      const updatedMessages = state.messages.map((msg) => {
        if (!msg.cards) return msg;
        const nextCards = msg.cards.map((card) => {
          if (card.type === 'ticket_draft') {
            return {
              type: 'ticket_draft' as const,
              payload: nextDraft,
            };
          }
          return card;
        });
        return { ...msg, cards: nextCards };
      });

      return {
        activeTicketDraft: nextDraft,
        messages: updatedMessages,
      };
    });
  },

  cancelTicketDraft: () => {
    set({
      ticketFlowStep: 'idle',
      activeTicketDraft: null,
    });
  },

  sendMessage: async (text) => {
    const trimmed = text.trim();
    if (!trimmed) return;

    abortActiveStream();
    const abortController = new AbortController();
    activeAbortController = abortController;

    const userMessage = createUserMessage(trimmed);
    const buddyMessage = createBuddyMessage();

    const currentFlowStep = get().ticketFlowStep;
    const currentDraft = get().activeTicketDraft;

    set((state) => ({
      messages: [...state.messages, userMessage, buddyMessage],
      draft: '',
      isStreaming: true,
    }));

    void queryClient.invalidateQueries({ queryKey: ['dialogs'] });

    try {
      await simulateSSEStream(
        trimmed,
        (event) => get().applySSEEvent(buddyMessage.id, event),
        abortController.signal,
        {
          flowStep: currentFlowStep,
          currentDraft,
          lastUserQuery: trimmed,
        },
      );
    } catch {
      return;
    } finally {
      if (activeAbortController === abortController) {
        set({ isStreaming: false });
        activeAbortController = null;
      }
    }
  },

  retryMessage: async (buddyMessageId) => {
    const { messages, sendMessage } = get();
    if (!buddyMessageId) {
      const lastUserMsg = [...messages].reverse().find((m) => m.sender === 'user');
      if (lastUserMsg) {
        await sendMessage(lastUserMsg.text);
      }
      return;
    }

    const index = messages.findIndex((m) => m.id === buddyMessageId);
    if (index <= 0) return;

    const previousUserMsg = messages[index - 1];
    if (previousUserMsg && previousUserMsg.sender === 'user') {
      set((state) => ({
        messages: state.messages.filter((m) => m.id !== buddyMessageId),
      }));
      await sendMessage(previousUserMsg.text);
    }
  },

  stopStreaming: () => {
    abortActiveStream();
    set({ isStreaming: false });
  },

  clearMessages: () => {
    abortActiveStream();
    set({ messages: [], isStreaming: false });
  },
}));
