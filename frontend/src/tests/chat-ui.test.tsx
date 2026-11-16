import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router';
import { ChatInput } from '@/widgets/ai-buddy-chat/ui/ChatInput';
import { BuddyChat } from '@/widgets/ai-buddy-chat/ui/BuddyChat';
import { ChatMessageItem } from '@/widgets/ai-buddy-chat/ui/ChatMessageItem';
import { ExecutionStages } from '@/widgets/ai-buddy-chat/ui/ExecutionStages';
import { ChatBackground } from '@/widgets/ai-buddy-chat/ui/ChatBackground';
import { useChatStore, type ChatMessage } from '@/entities/chat';
import { PROMPT_ACTIONS } from '@/widgets/ai-buddy-chat/constants/promptActions';
import { task } from './support';

const answer: ChatMessage = {
  id: 'a',
  role: 'assistant',
  content: 'Ответ',
  status: 'completed',
  stages: [],
};
const frame = (event: string, data: unknown) =>
  `event: ${event}\ndata: ${JSON.stringify(data)}\n\n`;
function stream(content = 'Ответ') {
  vi.stubGlobal(
    'fetch',
    vi.fn().mockImplementation(() =>
      Promise.resolve(
        new Response(frame('text', { text: content }) + frame('done', { content }), {
          headers: { 'Content-Type': 'text/event-stream' },
        }),
      ),
    ),
  );
}
it('expands input, preserves multiline text, sends trimmed text and clears the draft', async () => {
  const submit = vi.fn();
  render(<ChatInput onSubmit={submit} />);
  fireEvent.click(screen.getByRole('button', { name: 'Начать ввод' }));
  const input = screen.getByRole('textbox');
  expect(input).toHaveFocus();
  fireEvent.change(input, { target: { value: '  Привет\nМир  ' } });
  fireEvent.keyDown(input, { key: 'Enter', shiftKey: true });
  expect(submit).not.toHaveBeenCalled();
  fireEvent.keyDown(input, { key: 'Enter' });
  expect(submit).toHaveBeenCalledWith('Привет\nМир');
  expect(input).toHaveValue('');
  fireEvent.keyDown(input, { key: 'Enter' });
  expect(submit).toHaveBeenCalledTimes(1);
  fireEvent.keyDown(input, { key: 'Escape' });
  expect(screen.getByRole('button', { name: 'Начать ввод' })).toBeInTheDocument();
});
it('sends via the expanded button and collapses only after an empty blur outside the input', () => {
  const submit = vi.fn();
  render(<ChatInput onSubmit={submit} />);
  const input = screen.getByRole('textbox');
  fireEvent.focus(input);
  fireEvent.change(input, { target: { value: 'текст' } });
  fireEvent.blur(input, { relatedTarget: document.body });
  expect(screen.getByRole('button', { name: 'Отправить сообщение' })).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: 'Отправить сообщение' }));
  expect(submit).toHaveBeenCalledWith('текст');
  fireEvent.blur(input, {
    relatedTarget: screen.getByRole('button', { name: 'Отправить сообщение' }),
  });
  expect(useChatStore.getState().isExpanded).toBe(true);
  fireEvent.blur(input, { relatedTarget: document.body });
  expect(useChatStore.getState().isExpanded).toBe(false);
});
it.each([false, true])(
  'stops an active request using the square button (expanded=%s)',
  async (expanded) => {
    vi.stubGlobal(
      'fetch',
      vi.fn(
        (_url, options) =>
          new Promise((_resolve, reject) =>
            options.signal.addEventListener('abort', () =>
              reject(new DOMException('Aborted', 'AbortError')),
            ),
          ),
      ),
    );
    act(() => {
      void useChatStore.getState().sendMessage('Вопрос');
      useChatStore.getState().setIsExpanded(expanded);
    });
    render(<ChatInput />);
    const input = screen.getByRole('textbox');
    fireEvent.change(input, { target: { value: 'Черновик' } });
    fireEvent.keyDown(input, { key: 'Enter' });
    expect(useChatStore.getState().messages).toHaveLength(2);
    const stop = screen.getByRole('button', { name: 'Остановить ответ' });
    expect(stop.querySelector('.lucide-square')).toBeInTheDocument();
    fireEvent.click(stop);
    await waitFor(() => expect(useChatStore.getState().busy).toBe(false));
    expect(input).toHaveValue('Черновик');
  },
);
it.each(PROMPT_ACTIONS)('sends the exact quick request: $title', async (action) => {
  stream();
  render(
    <MemoryRouter>
      <BuddyChat />
    </MemoryRouter>,
  );
  fireEvent.click(
    screen.getByRole('button', { name: new RegExp(action.title.replace(/[.?]/g, '\\$&')) }),
  );
  await screen.findByText('Ответ');
  expect(useChatStore.getState().messages[0].content).toBe(action.promptText);
  expect(Element.prototype.scrollIntoView).toHaveBeenCalledWith({
    behavior: 'smooth',
    block: 'end',
  });
  const textarea = screen.getByRole('textbox');
  fireEvent.focus(textarea);
  const wrapper = textarea.closest('.max-w-\\[760px\\]');
  if (wrapper) {
    const event = new Event('transitionend', { bubbles: true });
    Object.defineProperty(event, 'propertyName', { value: 'height' });
    fireEvent(wrapper, event);
  }
});
it('copies, retries and reports clipboard failures without hiding the response', async () => {
  const writeText = vi.fn().mockResolvedValue(undefined);
  vi.stubGlobal('navigator', { clipboard: { writeText } });
  const retry = vi.fn();
  const view = render(
    <MemoryRouter>
      <ChatMessageItem message={answer} busy={false} onRetry={retry} />
    </MemoryRouter>,
  );
  fireEvent.click(screen.getByRole('button', { name: 'Копировать' }));
  await waitFor(() => expect(writeText).toHaveBeenCalledWith('Ответ'));
  fireEvent.click(screen.getByRole('button', { name: 'Повторить ответ' }));
  expect(retry).toHaveBeenCalledWith('a');
  view.unmount();
  writeText.mockRejectedValue(new Error('denied'));
  render(
    <MemoryRouter>
      <ChatMessageItem message={answer} busy={false} onRetry={retry} />
    </MemoryRouter>,
  );
  fireEvent.click(screen.getByRole('button', { name: 'Копировать' }));
  await screen.findByText('Не удалось скопировать текст');
  expect(screen.getByText('Ответ')).toBeInTheDocument();
});
it('renders task lists, empty tasks, proposal cards and streaming errors', () => {
  const view = render(
    <MemoryRouter>
      <ChatMessageItem
        message={{
          ...answer,
          tasks: [
            task,
            { ...task, id: 'done', status: 'completed' },
            { ...task, id: 'late', status: 'overdue' },
          ],
        }}
        busy={false}
        onRetry={() => {}}
      />
    </MemoryRouter>,
  );
  expect(screen.getAllByRole('heading', { name: task.description })).toHaveLength(3);
  view.rerender(
    <MemoryRouter>
      <ChatMessageItem
        message={{
          ...answer,
          tasks: [],
          ticket: { title: 'Тема', description: 'Текст' },
          task: { description: 'Задача', deadline: task.deadline },
          error: 'Прервано',
          status: 'error',
        }}
        busy={true}
        onRetry={() => {}}
      />
    </MemoryRouter>,
  );
  expect(screen.getByText('Задач пока нет')).toBeInTheDocument();
  expect(screen.getByText('Черновик тикета')).toBeInTheDocument();
  expect(screen.getByText('Черновик задачи')).toBeInTheDocument();
  expect(screen.getByRole('alert')).toHaveTextContent('Прервано');
  expect(screen.getByRole('button', { name: 'Повторить ответ' })).toBeDisabled();
});
it('keeps execution stages chronological and lets the user collapse completed steps', () => {
  const stages = [
    { id: '1', label: 'Поиск', status: 'completed' as const },
    { id: '2', label: 'Ответ', status: 'in_progress' as const },
  ];
  const view = render(<ExecutionStages stages={stages} />);
  expect(screen.getByText('Поиск')).toBeInTheDocument();
  expect(screen.getByText('Ответ')).toBeInTheDocument();
  view.rerender(<ExecutionStages stages={stages} isCompleted showSkeleton={false} />);
  fireEvent.click(screen.getByRole('button', { name: /Выполнено этапов: 2/ }));
  expect(screen.queryByText('Поиск')).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('button'));
  expect(screen.getByText('Поиск')).toBeInTheDocument();
  view.rerender(
    <ExecutionStages
      stages={[{ id: 'err', label: 'Сбой', status: 'error' }]}
      showSkeleton={false}
    />,
  );
  expect(screen.getByText('!')).toBeInTheDocument();
  view.rerender(
    <ExecutionStages
      stages={[{ id: 'one', label: 'Один этап', status: 'completed' }]}
      showSkeleton={false}
    />,
  );
  expect(screen.getByRole('button')).toHaveTextContent('Выполнен 1 этап');
  view.rerender(<ExecutionStages stages={[]} />);
  expect(screen.queryByRole('button')).not.toBeInTheDocument();
});
it('pauses decorations when hidden, resumes on visibility and removes its listener on unmount', () => {
  const remove = vi.spyOn(document, 'removeEventListener');
  const view = render(<ChatBackground />);
  const bg = view.container.firstElementChild!;
  Object.defineProperty(document, 'hidden', { configurable: true, value: true });
  fireEvent(document, new Event('visibilitychange'));
  expect(bg).toHaveAttribute('data-paused');
  Object.defineProperty(document, 'hidden', { configurable: true, value: false });
  fireEvent(document, new Event('visibilitychange'));
  expect(bg).not.toHaveAttribute('data-paused');
  view.unmount();
  expect(remove).toHaveBeenCalledWith('visibilitychange', expect.any(Function));
  Reflect.deleteProperty(document, 'hidden');
});
it('retries a response using its preceding user text and ignores invalid retries while busy', async () => {
  stream('Новый ответ');
  useChatStore.setState({
    messages: [{ ...answer, id: 'u', role: 'user', content: 'Исходный запрос' }, answer],
  });
  await act(async () => useChatStore.getState().retryMessage('a'));
  await waitFor(() => expect(useChatStore.getState().busy).toBe(false));
  expect(useChatStore.getState().messages.map((m) => m.content)).toEqual([
    'Исходный запрос',
    'Новый ответ',
  ]);
  const before = useChatStore.getState().messages;
  act(() => useChatStore.getState().retryMessage('missing'));
  expect(useChatStore.getState().messages).toBe(before);
  useChatStore.setState({ busy: true });
  act(() => {
    useChatStore.getState().retryMessage('a');
    void useChatStore.getState().sendMessage('ignored');
  });
  expect(useChatStore.getState().messages).toBe(before);
});
