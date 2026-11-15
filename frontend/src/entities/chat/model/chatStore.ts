import { create } from 'zustand';
import type { ChatMessage, SSEEvent } from './types';
import { simulateSSEStream } from '../api/chatStream';
import { createUserMessage, createBuddyMessage, updateMessagesWithEvent } from './helpers';

export interface ChatState {
  draft: string;
  isExpanded: boolean;
  messages: ChatMessage[];
  isStreaming: boolean;
}

export interface ChatActions {
  setDraft: (text: string) => void;
  setIsExpanded: (expanded: boolean) => void;
  applyPrompt: (promptText: string) => void;
  resetDraft: () => void;
  sendMessage: (text: string) => Promise<void>;
  retryMessage: (buddyMessageId?: string) => Promise<void>;
  applySSEEvent: (buddyMessageId: string, event: SSEEvent) => void;
  sendMentorTicket: (payload: { subject: string; message: string; mentorId: string }) => void;
  clearMessages: () => void;
  stopStreaming: () => void;
}

export type ChatStore = ChatState & ChatActions;

const initialChatState: ChatState = {
  draft: '',
  isExpanded: false,
  messages: [],
  isStreaming: false,
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

  applyPrompt: (promptText) => {
    set({ draft: promptText, isExpanded: true });
    get().sendMessage(promptText);
  },

  applySSEEvent: (buddyMessageId, event) => {
    set((state) => ({
      messages: updateMessagesWithEvent(state.messages, buddyMessageId, event),
      isStreaming: event.type === 'done' || event.type === 'error' ? false : state.isStreaming,
    }));
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
    }));
  },

  sendMessage: async (text) => {
    const trimmed = text.trim();
    if (!trimmed) return;

    abortActiveStream();
    const abortController = new AbortController();
    activeAbortController = abortController;

    const userMessage = createUserMessage(trimmed);
    const buddyMessage = createBuddyMessage();

    set((state) => ({
      messages: [...state.messages, userMessage, buddyMessage],
      draft: '',
      isStreaming: true,
    }));

    try {
      await simulateSSEStream(
        trimmed,
        (event) => get().applySSEEvent(buddyMessage.id, event),
        abortController.signal,
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
