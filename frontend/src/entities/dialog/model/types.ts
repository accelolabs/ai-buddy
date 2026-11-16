import type { ChatMessage } from '@/entities/chat/model/types';

export type DialogTopic = 'docs' | 'code' | 'chat' | 'other';

export interface Dialog {
  id: string;
  title: string;
  topic?: DialogTopic | string;
  createdAt: string;
  updatedAt: string;
}

export interface DialogsListResponse {
  dialogs: Dialog[];
}

export interface CreateDialogPayload {
  title?: string;
  topic?: DialogTopic | string;
}

export interface DialogMessagesResponse {
  dialogId: string;
  messages: ChatMessage[];
}
