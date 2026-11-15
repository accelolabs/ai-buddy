import { create } from 'zustand';

export interface TicketsUiState {
  copiedTicketId: string | null;
  filterStatus: 'all' | 'in_review' | 'resolved';
}

export interface TicketsUiActions {
  setCopiedTicketId: (ticketId: string | null) => void;
  setFilterStatus: (status: 'all' | 'in_review' | 'resolved') => void;
  copyTicketLink: (ticketId: string, url: string) => Promise<boolean>;
}

export type TicketsUiStore = TicketsUiState & TicketsUiActions;

export const useTicketsUiStore = create<TicketsUiStore>((set) => {
  let timer: ReturnType<typeof setTimeout> | null = null;

  return {
    copiedTicketId: null,
    filterStatus: 'all',

    setCopiedTicketId: (ticketId) => set({ copiedTicketId: ticketId }),
    setFilterStatus: (filterStatus) => set({ filterStatus }),

    copyTicketLink: async (ticketId, url) => {
      try {
        await navigator.clipboard.writeText(url);
        if (timer) clearTimeout(timer);
        set({ copiedTicketId: ticketId });
        timer = setTimeout(() => {
          set({ copiedTicketId: null });
        }, 2000);
        return true;
      } catch {
        return false;
      }
    },
  };
});
