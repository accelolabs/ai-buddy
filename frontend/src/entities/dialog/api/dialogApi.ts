import { request } from '@/shared/api';
import type { ChatMessage } from '@/entities/chat/model/types';
import type {
  Dialog,
  DialogsListResponse,
  CreateDialogPayload,
  DialogMessagesResponse,
} from '../model/types';
import mockDialogs from '../../../../../mock-server/dialogs.json';

const INITIAL_MOCK_DIALOGS: Dialog[] = (mockDialogs.dialogs || mockDialogs) as Dialog[];

const INITIAL_MOCK_MESSAGES: Record<string, ChatMessage[]> = {
  'dlg-001': [
    {
      id: 'msg-001-1',
      sender: 'user',
      text: 'Где найти регламент стажировки и чек-лист адаптации?',
      status: 'completed',
      createdAt: '2026-11-14T11:20:00Z',
    },
    {
      id: 'msg-001-2',
      sender: 'buddy',
      text: 'Привет! Регламент стажировки и чек-лист находятся в базе знаний Wiki команды онбординга. Основные этапы адаптации собраны в памятке стажера, а цели на первые недели зафиксированы в трекере.',
      status: 'completed',
      createdAt: '2026-11-14T11:20:05Z',
      cards: [
        {
          type: 'wiki_source',
          payload: {
            title: 'Регламент адаптации стажёров',
            section: '1. Общие положения и доступы',
            url: 'https://wiki.avito.ru/trainee/handbook#section-1',
            quote:
              'В первую неделю стажёр изучает регламент, настраивает окружение и согласует план с наставником.',
          },
        },
      ],
    },
  ],
  'dlg-002': [
    {
      id: 'msg-002-1',
      sender: 'user',
      text: 'При запуске docker-compose возникает ошибка: bind: address already in use для порта 5432.',
      status: 'completed',
      createdAt: '2026-11-14T09:15:00Z',
    },
    {
      id: 'msg-002-2',
      sender: 'buddy',
      text: 'Порт 5432 уже занят локальным процессом PostgreSQL. Вы можете либо остановить локальную службу postgres (`sudo systemctl stop postgresql`), либо переназначить внешний порт в `.env` на свободный (например, `5433:5432`).',
      status: 'completed',
      createdAt: '2026-11-14T09:15:06Z',
    },
  ],
  'dlg-003': [
    {
      id: 'msg-003-1',
      sender: 'user',
      text: 'Как сервис Buddy общается с Wiki, Tracker и Ticket?',
      status: 'completed',
      createdAt: '2026-11-13T15:00:00Z',
    },
    {
      id: 'msg-003-2',
      sender: 'buddy',
      text: 'Buddy — это Go-сервис, который использует MCP-сервер для вызова инструментов. Взаимодействие с Wiki, Tracker и Ticket происходит по защищённому REST API с передачей авторизационных заголовков и ID стажёра из доверенной сессии.',
      status: 'completed',
      createdAt: '2026-11-13T15:00:08Z',
    },
  ],
  'dlg-004': [
    {
      id: 'msg-004-1',
      sender: 'user',
      text: 'Какие основные доступы мне понадобятся в первую неделю?',
      status: 'completed',
      createdAt: '2026-11-13T11:00:00Z',
    },
    {
      id: 'msg-004-2',
      sender: 'buddy',
      text: 'В первую неделю вам понадобятся: учётная запись в Authentik, доступ к репозиториям проекта, командный чат и задачи в Tracker. Если какого-то доступа не хватает, создайте тикет наставнику.',
      status: 'completed',
      createdAt: '2026-11-13T11:00:05Z',
    },
  ],
};

let inMemoryDialogs: Dialog[] = [...INITIAL_MOCK_DIALOGS];
let inMemoryMessages: Record<string, ChatMessage[]> = { ...INITIAL_MOCK_MESSAGES };

export function resetMockDialogs() {
  inMemoryDialogs = [...INITIAL_MOCK_DIALOGS];
  inMemoryMessages = { ...INITIAL_MOCK_MESSAGES };
}

export function saveMockDialogMessages(dialogId: string, messages: ChatMessage[]) {
  inMemoryMessages[dialogId] = messages;
}

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
      return [...inMemoryDialogs];
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
      const newDialog: Dialog = {
        id: `dlg-${Date.now().toString().slice(-4)}`,
        title: payload.title || 'Новый диалог',
        topic: payload.topic || 'chat',
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      };
      inMemoryDialogs = [newDialog, ...inMemoryDialogs];
      inMemoryMessages[newDialog.id] = [];
      return newDialog;
    }
    throw error;
  }
}

export async function deleteDialogApi(id: string): Promise<void> {
  try {
    await request<void>(`/api/dialogs/${id}`, {
      method: 'DELETE',
    });
  } catch (error) {
    if (import.meta.env.DEV) {
      inMemoryDialogs = inMemoryDialogs.filter((dialog) => dialog.id !== id);
      delete inMemoryMessages[id];
      return;
    }
    throw error;
  }
}

export async function fetchDialogMessages(dialogId: string): Promise<ChatMessage[]> {
  try {
    const data = await request<DialogMessagesResponse | ChatMessage[]>(
      `/api/dialogs/${dialogId}/messages`,
      {
        method: 'GET',
      },
    );
    if (Array.isArray(data)) return data;
    if (data && typeof data === 'object' && 'messages' in data && Array.isArray(data.messages)) {
      return data.messages;
    }
    return [];
  } catch (error) {
    if (import.meta.env.DEV) {
      return inMemoryMessages[dialogId] ? [...inMemoryMessages[dialogId]] : [];
    }
    throw error;
  }
}
