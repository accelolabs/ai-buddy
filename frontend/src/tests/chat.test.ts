import { afterEach, describe, expect, it, vi } from 'vitest';
import { parseSSE, streamChat } from '@/entities/chat/api/chatStream';
import { useChatStore } from '@/entities/chat';
import { changeUser } from '@/shared/api/session';
import { queryClient } from '@/shared/api';

function body(text: string, size = 1) {
  const bytes = new TextEncoder().encode(text);
  return new ReadableStream<Uint8Array>({
    start(controller) {
      for (let i = 0; i < bytes.length; i += size) controller.enqueue(bytes.slice(i, i + size));
      controller.close();
    },
  });
}
function response(text: string) {
  return new Response(body(text), { headers: { 'Content-Type': 'text/event-stream' } });
}
const frame = (type: string, data: unknown) => `event: ${type}\ndata: ${JSON.stringify(data)}\n\n`;
afterEach(() => {
  changeUser(null);
  vi.unstubAllGlobals();
});

describe('SSE and in-memory chat', () => {
  it('reads split UTF-8, CRLF, comments and multiline data', async () => {
    const events: unknown[] = [];
    await parseSSE(
      body(': keepalive\r\nevent: text\r\ndata: {"text":\r\ndata: "Привет"}\r\n\r\n'),
      (event) => events.push(event),
    );
    expect(events).toEqual([{ type: 'text', data: { text: 'Привет' } }]);
  });
  it('fails on incomplete streams and reports SSE errors', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(frame('text', { text: 'Часть' }))));
    await expect(
      streamChat([{ role: 'user', content: 'Вопрос' }], () => {}, new AbortController().signal),
    ).rejects.toThrow('прерван');
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(response(frame('error', { error: 'model unavailable' }))),
    );
    await expect(streamChat([], () => {}, new AbortController().signal)).rejects.toThrow(
      'model unavailable',
    );
  });
  it('merges all cards from done without duplicates and sends only text history', async () => {
    changeUser('intern');
    const ticket = { title: 'Тема', description: 'Описание' };
    const task = { description: 'Изучить', deadline: '2027-01-01' };
    const completion = {
      ...task,
      id: 'task-1',
      user_id: 'intern',
      status: 'pending',
      created_at: '2027-01-01T10:00:00+03:00',
    };
    const source = {
      document_id: 'wiki-1',
      anchor: 'раздел',
      title: 'Статья',
      quote: 'Цитата',
      ordinal: 0,
    };
    const fetch = vi.fn().mockImplementation(() =>
      Promise.resolve(
        response(
          frame('state', { state: 'thinking' }) +
            frame('text', { text: 'Ответ' }) +
            frame('source', source) +
            frame('ticket', ticket) +
            frame('task', task) +
            frame('tasks', [completion]) +
            frame('task_completion', completion) +
            frame('done', {
              content: 'Ответ',
              source,
              ticket,
              task,
              tasks: [completion],
              task_completion: completion,
            }),
        ),
      ),
    );
    vi.stubGlobal('fetch', fetch);
    await useChatStore.getState().sendMessage('Вопрос');
    expect(useChatStore.getState().messages).toHaveLength(2);
    expect(useChatStore.getState().messages[1]).toMatchObject({
      content: 'Ответ',
      status: 'completed',
      ticket,
      task,
      source,
      tasks: [completion],
      task_completion: completion,
    });
    await useChatStore.getState().sendMessage('да');
    expect(fetch).toHaveBeenCalledTimes(2);
    expect(fetch.mock.calls.every(([url]) => url === '/api/buddy/messages')).toBe(true);
    expect(JSON.parse(fetch.mock.calls[1][1].body)).toEqual({
      messages: [
        { role: 'user', content: 'Вопрос' },
        {
          role: 'assistant',
          content: `Ответ\n\nЗадачи в карточке:\n${JSON.stringify([{ id: completion.id, description: completion.description, status: completion.status }])}`,
        },
        { role: 'user', content: 'да' },
      ],
    });
    useChatStore.getState().startNewChat();
    expect(useChatStore.getState().messages).toEqual([]);
  });
  it.each(['tasks', 'task_completion'] as const)(
    'passes actual task IDs from %s cards to the next model request',
    async (card) => {
      const task = {
        id: 'ZX66BRN6SKY6CMOZN6BBEZ3O3K',
        user_id: 'intern',
        description: 'Украсть Фобос',
        status: 'pending' as const,
        deadline: '2027-01-01',
        created_at: '',
      };
      const fetch = vi
        .fn()
        .mockResolvedValue(
          response(frame('done', { content: 'Подтвердите кнопкой', task_completion: task })),
        );
      vi.stubGlobal('fetch', fetch);
      useChatStore.setState({
        messages: [
          {
            id: 'card',
            role: 'assistant',
            content: '',
            status: 'completed',
            stages: [],
            [card]: card === 'tasks' ? [task] : task,
          },
        ],
      });
      await useChatStore.getState().sendMessage('Я украл Фобос');
      const messages = JSON.parse(fetch.mock.calls[0][1].body).messages;
      expect(messages).toEqual([
        {
          role: 'assistant',
          content: `\n\nЗадачи в карточке:\n${JSON.stringify([{ id: task.id, description: task.description, status: task.status }])}`,
        },
        { role: 'user', content: 'Я украл Фобос' },
      ]);
      expect(useChatStore.getState().messages[0].content).toBe('');
      expect(useChatStore.getState().messages.at(-1)?.task_completion?.id).toBe(task.id);
      expect(fetch).toHaveBeenCalledTimes(1);
      expect(fetch.mock.calls[0][0]).toBe('/api/buddy/messages');
    },
  );
  it('appends repeated states and tool calls below the completed steps', async () => {
    const states = [
      { state: 'thinking' },
      { state: 'progress', progress: 'search_documents' },
      { state: 'thinking' },
      { state: 'progress', progress: 'attach_source' },
      { state: 'thinking' },
      { state: 'answering' },
    ];
    const snapshots: { label: string; status: string }[][] = [];
    const unsubscribe = useChatStore.subscribe((s) => {
      const stages = s.messages.at(-1)?.stages;
      if (stages?.at(-1)?.status === 'in_progress') snapshots.push(stages);
    });
    vi.stubGlobal(
      'fetch',
      vi
        .fn()
        .mockResolvedValue(
          response(
            states.map((state) => frame('state', state)).join('') +
              frame('done', { content: 'Ответ' }),
          ),
        ),
    );
    try {
      await useChatStore.getState().sendMessage('Вопрос');
    } finally {
      unsubscribe();
    }
    expect(snapshots.map((stages) => stages.length)).toEqual([1, 2, 3, 4, 5, 6]);
    snapshots.forEach((stages) => {
      expect(stages.slice(0, -1).every((stage) => stage.status === 'completed')).toBe(true);
      expect(stages.at(-1)?.status).toBe('in_progress');
    });
    const stages = useChatStore.getState().messages[1].stages;
    expect(stages.map((stage) => stage.label)).toEqual([
      'Обдумываю запрос',
      'Выполняю действие · Ищу в базе знаний',
      'Обдумываю запрос',
      'Выполняю действие · Добавляю источник',
      'Обдумываю запрос',
      'Формулирую ответ',
    ]);
    expect(new Set(stages.map((stage) => stage.id)).size).toBe(6);
    expect(stages.every((stage) => stage.status === 'completed')).toBe(true);
  });
  it.each([{}, { source: null }])(
    'keeps the streamed source when done has no attachment: %j',
    async (final) => {
      const source = {
        document_id: 'wiki-1',
        ordinal: 1,
        title: 'Статья',
        anchor: 'раздел',
        quote: 'Цитата',
      };
      vi.stubGlobal(
        'fetch',
        vi
          .fn()
          .mockResolvedValue(
            response(frame('source', source) + frame('done', { content: 'Ответ', ...final })),
          ),
      );
      await useChatStore.getState().sendMessage('Вопрос');
      expect(useChatStore.getState().messages[1].source).toEqual(source);
    },
  );
  it('clears state and cache, aborting all requests on user changes', async () => {
    changeUser('first');
    queryClient.setQueryData(['tasks'], ['private']);
    useChatStore.setState({ draft: 'private' });
    let signal: AbortSignal | undefined;
    vi.stubGlobal(
      'fetch',
      vi.fn(
        (_url, options) =>
          new Promise((_resolve, reject) => {
            signal = options.signal;
            signal?.addEventListener('abort', () =>
              reject(new DOMException('Aborted', 'AbortError')),
            );
          }),
      ),
    );
    const sending = useChatStore.getState().sendMessage('Личное');
    changeUser('second');
    await sending;
    expect(signal?.aborted).toBe(true);
    expect(useChatStore.getState()).toMatchObject({ messages: [], draft: '', busy: false });
    expect(queryClient.getQueryData(['tasks'])).toBeUndefined();
    expect(document.cookie).toContain('demo_user=second');
  });
  it('finishes stopped streams with an error, allowing retry', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(
        (_url, options) =>
          new Promise((_resolve, reject) => {
            options.signal.addEventListener('abort', () =>
              reject(new DOMException('Aborted', 'AbortError')),
            );
          }),
      ),
    );
    const sending = useChatStore.getState().sendMessage('Вопрос');
    useChatStore.getState().stop();
    await sending;
    expect(useChatStore.getState()).toMatchObject({ busy: false });
    expect(useChatStore.getState().messages[1]).toMatchObject({
      status: 'error',
      error: 'Ответ остановлен',
    });
  });
});
