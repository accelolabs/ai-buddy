import { create } from 'zustand';

export interface ChatStore {
  draft: string;
  isExpanded: boolean;
  setDraft: (text: string) => void;
  setIsExpanded: (expanded: boolean) => void;
  applyPrompt: (promptText: string) => void;
  resetDraft: () => void;
}

export const useChatStore = create<ChatStore>((set) => ({
  draft: '',
  isExpanded: false,
  setDraft: (draft) => set({ draft }),
  setIsExpanded: (isExpanded) => set({ isExpanded }),
  applyPrompt: (promptText) => set({ draft: promptText, isExpanded: true }),
  resetDraft: () => set({ draft: '' }),
}));
