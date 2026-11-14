import { useEffect, useRef } from 'react';
import type { ChatMessage, MentorPayload } from '@/entities/chat';
import { ChatMessageItem } from './ChatMessageItem';

export interface MessageListProps {
  messages: ChatMessage[];
  onRetry?: (messageId: string) => void;
  onContactMentor?: (mentor: MentorPayload) => void;
}

export function MessageList({ messages, onRetry, onContactMentor }: MessageListProps) {
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages]);

  return (
    <div className="flex w-full flex-1 flex-col overflow-y-auto px-4 py-6 sm:px-6">
      <div className="mx-auto flex w-full max-w-[760px] flex-col gap-2">
        {messages.map((message) => (
          <ChatMessageItem
            key={message.id}
            message={message}
            onRetry={onRetry}
            onContactMentor={onContactMentor}
          />
        ))}
        <div ref={bottomRef} className="h-4" />
      </div>
    </div>
  );
}

export default MessageList;
