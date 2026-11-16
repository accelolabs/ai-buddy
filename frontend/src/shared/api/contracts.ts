export interface TaskDraft {
  description: string;
  /** Calendar date in YYYY-MM-DD format. */
  deadline: string;
}
export interface Task extends TaskDraft {
  id: string;
  user_id: string;
  status: 'pending' | 'overdue' | 'completed';
  created_at: string;
}
export interface TicketDraft {
  recipient?: { id: string; name: string };
  title: string;
  description: string;
}
export interface Ticket extends TicketDraft {
  id: string;
  user_id: string;
  created_at: string;
}
export interface Source {
  document_id: string;
  ordinal: number;
  title: string;
  anchor: string;
  quote: string;
}
export interface Article {
  id: string;
  title: string;
  team: string;
}
export const taskStatus = (status: Task['status']) =>
  ({ pending: 'Впереди', overdue: 'Просрочено', completed: 'Выполнено' })[status];
export const dateTime = (value: string) => new Date(value).toLocaleString('ru-RU');

export const calendarDate = (value: string) => value.split('-').reverse().join('.');
