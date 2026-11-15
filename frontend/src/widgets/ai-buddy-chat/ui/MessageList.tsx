import { useEffect, useRef } from 'react';
import type { ChatMessage, MentorPayload, TicketDraftPayload } from '@/entities/chat';
import { ChatMessageItem } from './ChatMessageItem';

export interface MessageListProps {
  messages: ChatMessage[];
  onRetry?: (messageId: string) => void;
  onContactMentor?: (mentor: MentorPayload) => void;
  onApproveDraft?: (draft: TicketDraftPayload) => void;
  onUpdateDraft?: (updated: Partial<TicketDraftPayload>) => void;
}

export function MessageList({
  messages,
  onRetry,
  onContactMentor,
  onApproveDraft,
  onUpdateDraft,
}: MessageListProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const bottomRef = useRef<HTMLDivElement>(null);
  const prevLengthRef = useRef(messages.length);
  const isAutoScrollEnabledRef = useRef(true);

  const handleScroll = () => {
    const container = containerRef.current;
    if (!container) return;
    const { scrollTop, scrollHeight, clientHeight } = container;
    const isNearBottom = scrollHeight - scrollTop - clientHeight < 100;
    isAutoScrollEnabledRef.current = isNearBottom;
  };

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const prevLength = prevLengthRef.current;
    const currentLength = messages.length;
    prevLengthRef.current = currentLength;

    if (currentLength > prevLength) {
      isAutoScrollEnabledRef.current = true;
      bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
      return;
    }

    if (isAutoScrollEnabledRef.current) {
      container.scrollTop = container.scrollHeight;
    }
  }, [messages]);

  return (
    <div
      ref={containerRef}
      onScroll={handleScroll}
      className="flex w-full flex-1 flex-col overflow-y-auto px-4 py-6 sm:px-6"
    >
      <div className="mx-auto flex w-full max-w-[760px] flex-col gap-2">
        {messages.map((message) => (
          <ChatMessageItem
            key={message.id}
            message={message}
            onRetry={onRetry}
            onContactMentor={onContactMentor}
            onApproveDraft={onApproveDraft}
            onUpdateDraft={onUpdateDraft}
          />
        ))}
        <div ref={bottomRef} className="h-4" />
      </div>
    </div>
  );
}

export default MessageList;
