import { request } from '@/shared/api';
import type { Dialog, DialogsListResponse, CreateDialogPayload } from '../model/types';
import mockDialogs from '../../../../../mock-server/dialogs.json';

export async function fetchDialogs(): Promise<Dialog[]> {
  try {
    const data = await request<DialogsListResponse | Dialog[]>('/api/dialogs', {
      method: 'GET',
    });
    if (Array.isArray(data)) return data;
    if (data && typeof data === 'object' && 'dialogs' in data && Array.isArray(data.dialogs)) {
      return data.dialogs;
    }
    return [];
  } catch (error) {
    if (import.meta.env.DEV) {
      console.warn('API /api/dialogs недоступен, используется mock-server:', error);
      return (mockDialogs.dialogs || mockDialogs) as Dialog[];
    }
    throw error;
  }
}

export async function createDialogApi(payload: CreateDialogPayload): Promise<Dialog> {
  try {
    return await request<Dialog>('/api/dialogs', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  } catch (error) {
    if (import.meta.env.DEV) {
      console.warn('POST /api/dialogs недоступен, эмулируем создание диалога в DEV:', error);
      const newDialog: Dialog = {
        id: `dlg-${Date.now().toString().slice(-4)}`,
        title: payload.title || 'Новый диалог',
        topic: payload.topic || 'chat',
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      };
      return newDialog;
    }
    throw error;
  }
}
