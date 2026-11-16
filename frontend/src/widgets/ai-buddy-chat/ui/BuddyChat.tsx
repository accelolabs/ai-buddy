import { useEffect, useRef } from 'react';
import { PromptButton, Picture } from '@/shared/ui';
import chipLogo from '@/assets/chip-logo.svg';
import { useChatStore } from '@/entities/chat';
import { PROMPT_ACTIONS } from '../constants/promptActions';
import { ChatInput } from './ChatInput';
import { ChatMessageItem } from './ChatMessageItem';

export function BuddyChat() {
  const { messages, busy, retryMessage, applyPrompt, isExpanded } = useChatStore();
  const input = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (messages.length || isExpanded) {
      input.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
    }
  }, [messages, isExpanded]);
  return (
    <div className="relative z-10 flex flex-1 flex-col">
      {messages.length ? (
        <div className="mx-auto w-full max-w-[760px] flex-1 px-4 pb-6">
          {messages.map((message) => (
            <ChatMessageItem
              key={message.id}
              message={message}
              onRetry={retryMessage}
              busy={busy}
            />
          ))}
        </div>
      ) : (
        <div className="flex flex-1 flex-col items-center justify-center gap-8 px-4 py-12">
          <Picture src={chipLogo} alt="AI-buddy" className="h-14 w-14" />
          <div className="text-center">
            <h1 className="text-3xl font-extrabold text-[#0F172A] sm:text-4xl">
              Привет! Я — твой AI-buddy.
            </h1>
            <p className="mt-2 text-xl text-[#64748B]">Чем помочь сегодня?</p>
          </div>
          <div className="grid w-full max-w-[712px] gap-4 sm:grid-cols-2">
            {PROMPT_ACTIONS.map((action) => (
              <PromptButton
                key={action.title}
                title={action.title}
                subtitle={action.subtitle}
                onClick={() => applyPrompt(action.promptText)}
              />
            ))}
          </div>
        </div>
      )}
      <div
        ref={input}
        onTransitionEnd={(event) => {
          if (event.propertyName === 'height') {
            input.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
          }
        }}
        className="mx-auto w-full max-w-[760px] px-4 pb-6"
      >
        <ChatInput />
      </div>
    </div>
  );
}
