import { create } from 'zustand';
import type { ChatMessage, SSEEvent, TicketDraftPayload, TicketFlowStep } from './types';
import { simulateSSEStream } from '../api/chatStream';
import { createTicket } from '../api/ticketApi';
import { createUserMessage, createBuddyMessage, updateMessagesWithEvent } from './helpers';
import { queryClient } from '@/shared/api';
import {
  createDialogApi,
  saveMockDialogMessages,
  type Dialog,
} from '@/entities/dialog';

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
  setMessages: (messages: ChatMessage[]) => void;
  loadDialog: (dialogId: string, messages?: ChatMessage[]) => void;
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

function generateDialogTitle(text: string): string {
  const clean = text.replace(/\s+/g, ' ').trim();
  if (clean.length <= 40) return clean;
  return clean.slice(0, 40).trim() + '...';
}

function detectTopic(text: string): 'docs' | 'code' | 'chat' {
  const lower = text.toLowerCase();
  if (
    lower.includes('документ') ||
    lower.includes('вики') ||
    lower.includes('wiki') ||
    lower.includes('регламент') ||
    lower.includes('чек-лист') ||
    lower.includes('памятк')
  ) {
    return 'docs';
  }
  if (
    lower.includes('docker') ||
    lower.includes('докер') ||
    lower.includes('код') ||
    lower.includes('порт') ||
    lower.includes('ошибк') ||
    lower.includes('баг') ||
    lower.includes('компил') ||
    lower.includes('сборк') ||
    lower.includes('тест')
  ) {
    return 'code';
  }
  return 'chat';
}

export const useChatStore = create<ChatStore>((set, get) => ({
  ...initialChatState,

  setDraft: (draft) => set({ draft }),
  setIsExpanded: (isExpanded) => set({ isExpanded }),
  resetDraft: () => set({ draft: '' }),
  setActiveDialogId: (activeDialogId) => set({ activeDialogId }),
  setMessages: (messages) => set({ messages }),

  loadDialog: (dialogId, messages = []) => {
    abortActiveStream();
    set({
      activeDialogId: dialogId,
      messages,
      draft: '',
      isStreaming: false,
      ticketFlowStep: 'idle',
      activeTicketDraft: null,
    });
  },

  startNewChat: () => {
    abortActiveStream();
    set({
      activeDialogId: null,
      messages: [],
      draft: '',
      isExpanded: false,
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

      const nextMessages = updateMessagesWithEvent(state.messages, buddyMessageId, event);

      if (state.activeDialogId) {
        saveMockDialogMessages(state.activeDialogId, nextMessages);
      }

      return {
        messages: nextMessages,
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
    set((state) => {
      const nextMessages = [...state.messages, confirmationMsg];
      if (state.activeDialogId) {
        saveMockDialogMessages(state.activeDialogId, nextMessages);
      }
      return {
        messages: nextMessages,
        ticketFlowStep: 'idle',
        activeTicketDraft: null,
      };
    });
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

        if (state.activeDialogId) {
          saveMockDialogMessages(state.activeDialogId, updatedMessages);
        }

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

      if (state.activeDialogId) {
        saveMockDialogMessages(state.activeDialogId, updatedMessages);
      }

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

    let activeId = get().activeDialogId;
    if (!activeId) {
      try {
        const newDialog = await createDialogApi({
          title: generateDialogTitle(trimmed),
          topic: detectTopic(trimmed),
        });
        activeId = newDialog.id;
        set({ activeDialogId: activeId });

        queryClient.setQueryData<Dialog[]>(['dialogs', 'list'], (old) => {
          return old ? [newDialog, ...old] : [newDialog];
        });
        queryClient.invalidateQueries({ queryKey: ['dialogs'] });
      } catch (err) {
        console.error('Failed to auto-create dialog for new message:', err);
      }
    }

    const userMessage = createUserMessage(trimmed);
    const buddyMessage = createBuddyMessage();

    const currentFlowStep = get().ticketFlowStep;
    const currentDraft = get().activeTicketDraft;

    const nextMessages = [...get().messages, userMessage, buddyMessage];
    if (activeId) {
      saveMockDialogMessages(activeId, nextMessages);
    }

    set({
      messages: nextMessages,
      draft: '',
      isStreaming: true,
    });

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
      const filtered = messages.filter((m) => m.id !== buddyMessageId);
      if (get().activeDialogId) {
        saveMockDialogMessages(get().activeDialogId!, filtered);
      }
      set({ messages: filtered });
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
