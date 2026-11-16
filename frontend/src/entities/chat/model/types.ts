import type { Source, Task, TaskDraft, TicketDraft } from '@/shared/api/contracts';
export interface MessageStage {
  id: string;
  label: string;
  status: 'in_progress' | 'completed' | 'error';
}
export interface Answer {
  source?: Source | null;
  ticket?: TicketDraft | null;
  task?: TaskDraft | null;
  tasks?: Task[] | null;
  task_completion?: Task | null;
}
export interface ChatMessage extends Answer {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  status: 'in_progress' | 'completed' | 'error';
  stages: MessageStage[];
  error?: string;
}
export type SSEEvent =
  | { type: 'state'; data: { state: string; progress?: string } }
  | { type: 'text'; data: { text: string } }
  | { type: 'source'; data: Source }
  | { type: 'ticket'; data: TicketDraft }
  | { type: 'task'; data: TaskDraft }
  | { type: 'tasks'; data: Task[] }
  | { type: 'task_completion'; data: Task }
  | { type: 'done'; data: Answer & { content: string } }
  | { type: 'error'; data: { error: string } };
