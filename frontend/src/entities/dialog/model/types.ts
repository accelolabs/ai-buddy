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
