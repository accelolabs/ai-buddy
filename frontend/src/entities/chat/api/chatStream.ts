import { fetchApi } from '@/shared/api/httpClient';
import type { SSEEvent } from '../model/types';

// Keep incomplete lines and frames across arbitrary byte chunks (including UTF-8).
export async function parseSSE(
  body: ReadableStream<Uint8Array>,
  onEvent: (event: SSEEvent) => void,
) {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let pending = '';
  let event = '';
  let data: string[] = [];
  const line = (value: string) => {
    if (!value) {
      if (data.length)
        onEvent({ type: event || 'message', data: JSON.parse(data.join('\n')) } as SSEEvent);
      event = '';
      data = [];
    } else if (value.startsWith('event:')) event = value.slice(6).trimStart();
    else if (value.startsWith('data:')) data.push(value.slice(5).replace(/^ /, ''));
  };
  try {
    while (true) {
      const { value, done } = await reader.read();
      pending += decoder.decode(value, { stream: !done });
      let index: number;
      while ((index = pending.indexOf('\n')) >= 0) {
        line(pending.slice(0, index).replace(/\r$/, ''));
        pending = pending.slice(index + 1);
      }
      if (done) break;
    }
  } finally {
    await reader.cancel().catch(() => {});
    reader.releaseLock();
  }
}

export async function streamChat(
  messages: { role: string; content: string }[],
  onEvent: (event: SSEEvent) => void,
  signal: AbortSignal,
) {
  const response = await fetchApi('/api/buddy/messages', {
    method: 'POST',
    body: JSON.stringify({ messages }),
    signal,
  });
  if (!response.body || !response.headers.get('Content-Type')?.includes('text/event-stream'))
    throw new Error('Сервер не вернул поток ответа');
  let done = false;
  await parseSSE(response.body, (event) => {
    signal.throwIfAborted();
    if (event.type === 'error') throw new Error(event.data.error);
    if (event.type === 'done') done = true;
    onEvent(event);
  });
  if (!done) throw new Error('Поток ответа прерван. Повторите запрос.');
}
