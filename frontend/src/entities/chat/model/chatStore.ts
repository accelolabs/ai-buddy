import { create } from 'zustand';
import { useSession } from '@/shared/api/session';
import { streamChat } from '../api/chatStream';
import type { ChatMessage, SSEEvent } from './types';

export interface ChatStore {
  messages: ChatMessage[];
  draft: string;
  isExpanded: boolean;
  busy: boolean;
  setDraft: (draft: string) => void;
  setIsExpanded: (isExpanded: boolean) => void;
  resetDraft: () => void;
  sendMessage: (text: string) => Promise<void>;
  applyPrompt: (text: string) => void;
  retryMessage: (id: string) => void;
  startNewChat: () => void;
  stop: () => void;
}
let controller: AbortController | null = null;
export const useChatStore = create<ChatStore>((set, get) => ({
  messages: [],
  draft: '',
  isExpanded: false,
  busy: false,
  setDraft: (draft) => set({ draft }),
  setIsExpanded: (isExpanded) => set({ isExpanded }),
  resetDraft: () => set({ draft: '' }),
  startNewChat: () => {
    controller?.abort();
    controller = null;
    set({ messages: [], draft: '', busy: false, isExpanded: false });
  },
  stop: () => {
    controller?.abort();
  },
  applyPrompt: (text) => {
    void get().sendMessage(text);
  },
  retryMessage: (id) => {
    if (get().busy) return;
    const index = get().messages.findIndex((m) => m.id === id);
    const previous = get().messages[index - 1];
    if (previous?.role !== 'user') return;
    set({ messages: get().messages.slice(0, index - 1) });
    void get().sendMessage(previous.content);
  },
  sendMessage: async (text) => {
    if (get().busy || !text.trim()) return;
    const message = (role: ChatMessage['role'], content: string): ChatMessage => ({
      id: crypto.randomUUID(),
      role,
      content,
      status: role === 'user' ? 'completed' : 'in_progress',
      stages: [],
    });
    const history = [...get().messages, message('user', text.trim())];
    const reply = message('assistant', '');
    const active = new AbortController();
    controller = active;
    set({ messages: [...history, reply], busy: true, draft: '' });
    const update = (fn: (m: ChatMessage) => ChatMessage) => {
      if (controller !== active) return;
      set((s) => ({ messages: s.messages.map((m) => (m.id === reply.id ? fn(m) : m)) }));
    };
    const onEvent = (event: SSEEvent) =>
      update((m) => {
        if (event.type === 'state') {
          const { state, progress } = event.data;
          const stages: ChatMessage['stages'] = m.stages.map((s) => ({
            ...s,
            status: 'completed' as const,
          }));
          const stage = {
            id: state,
            label:
              ({
                thinking: 'Обдумываю запрос',
                answering: 'Формулирую ответ',
                progress: 'Выполняю действие',
              }[state] || state) +
              (progress
                ? ` · ${{ search_documents: 'Ищу в базе знаний', attach_source: 'Добавляю источник', list_tasks: 'Загружаю задачи', prepare_task: 'Готовлю задачу', prepare_ticket: 'Готовлю обращение', prepare_task_completion: 'Готовлю завершение задачи' }[progress] || 'Обрабатываю запрос'}`
                : ''),
            status: 'in_progress' as const,
          };
          const index = stages.findIndex((s) => s.id === state);
          if (index >= 0) stages[index] = stage;
          else stages.push(stage);
          return { ...m, stages };
        }
        if (event.type === 'text') return { ...m, content: m.content + event.data.text };
        if (event.type === 'done')
          return {
            ...m,
            ...event.data,
            status: 'completed',
            stages: m.stages.map((s) => ({ ...s, status: 'completed' })),
          };
        if (event.type === 'error') return m;
        return { ...m, [event.type]: event.data };
      });
    try {
      await streamChat(
        history
          .filter((m) => m.content.trim() && (m.role === 'user' || m.status === 'completed'))
          .map(({ role, content }) => ({ role, content })),
        onEvent,
        active.signal,
      );
    } catch (error) {
      update((m) => ({
        ...m,
        status: 'error',
        error: active.signal.aborted
          ? 'Ответ остановлен'
          : error instanceof Error
            ? error.message
            : String(error),
        stages: m.stages.map((s) => (s.status === 'in_progress' ? { ...s, status: 'error' } : s)),
      }));
    } finally {
      if (controller === active) {
        controller = null;
        set({ busy: false });
      }
    }
  },
}));
useSession.subscribe(() => useChatStore.getState().startNewChat());
