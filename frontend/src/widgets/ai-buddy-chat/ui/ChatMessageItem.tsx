import { memo, useState } from 'react';
import { Copy, Check, RotateCcw } from 'lucide-react';
import type { ChatMessage, MentorPayload, TicketDraftPayload } from '@/entities/chat';
import { BuddyAvatar } from './BuddyAvatar';
import { ExecutionStages } from './ExecutionStages';
import { SourceCard } from './SourceCard';
import { MentorCard } from './MentorCard';
import { TicketDraftCard } from './TicketDraftCard';
import { TicketSuccessCard } from './TicketSuccessCard';

export interface ChatMessageItemProps {
  message: ChatMessage;
  onRetry?: (messageId: string) => void;
  onContactMentor?: (mentor: MentorPayload) => void;
  onApproveDraft?: (draft: TicketDraftPayload) => void;
  onUpdateDraft?: (updated: Partial<TicketDraftPayload>) => void;
}

export const ChatMessageItem = memo(function ChatMessageItem({
  message,
  onRetry,
  onContactMentor,
  onApproveDraft,
  onUpdateDraft,
}: ChatMessageItemProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    if (!message.text) return;
    try {
      await navigator.clipboard.writeText(message.text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      void 0;
    }
  };

  if (message.sender === 'user') {
    return (
      <div className="flex w-full justify-end py-2">
        <div className="max-w-[85%] rounded-[20px] rounded-tr-[4px] bg-[#F1F5F9] px-5 py-3 text-[15px] leading-relaxed text-[#0F172A] shadow-xs sm:max-w-[70%]">
          {message.text}
        </div>
      </div>
    );
  }

  const isInProgress = message.status === 'in_progress';
  const hasStages = message.stages && message.stages.length > 0;
  const hasText = Boolean(message.text);
  const hasCards = message.cards && message.cards.length > 0;

  return (
    <div className="flex w-full gap-3 py-3">
      <BuddyAvatar className="h-8 w-8 mt-0.5" />

      <div className="flex flex-1 flex-col gap-3.5">
        {hasStages && (
          <ExecutionStages
            stages={message.stages!}
            showSkeleton={isInProgress && !hasText}
            isCompleted={message.status === 'completed'}
          />
        )}

        {hasText && (
          <div className="text-[15px] leading-relaxed text-[#0F172A]">
            <span className="whitespace-pre-wrap">{message.text}</span>
            {isInProgress && (
              <span
                className="ml-1 inline-block h-4 w-1.5 animate-pulse rounded-xs bg-[#00AAFF] align-middle"
                aria-label="Генерация текста"
              />
            )}
          </div>
        )}

        {hasCards && (
          <div className="flex flex-col gap-3">
            {message.cards!.map((card, idx) => {
              if (card.type === 'wiki_source') {
                return <SourceCard key={idx} source={card.payload} />;
              }
              if (card.type === 'mentor_fallback') {
                return <MentorCard key={idx} mentor={card.payload} onContact={onContactMentor} />;
              }
              if (card.type === 'ticket_draft') {
                return (
                  <TicketDraftCard
                    key={idx}
                    draft={card.payload}
                    onApprove={onApproveDraft}
                    onUpdate={onUpdateDraft}
                  />
                );
              }
              if (card.type === 'ticket_success') {
                return <TicketSuccessCard key={idx} ticket={card.payload} />;
              }
              return null;
            })}
          </div>
        )}

        {message.status === 'error' && (
          <div className="flex flex-col gap-2 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-800">
            <p>{message.error || 'Произошла ошибка при обработке запроса.'}</p>
            {onRetry && (
              <button
                type="button"
                onClick={() => onRetry(message.id)}
                className="self-start rounded-lg bg-red-600 px-3 py-1.5 text-xs font-semibold text-white transition hover:bg-red-700"
              >
                Повторить
              </button>
            )}
          </div>
        )}

        {message.status === 'completed' && hasText && (
          <div className="flex items-center gap-2 pt-1 text-[#64748B]">
            <button
              type="button"
              onClick={handleCopy}
              className="flex h-7 w-7 items-center justify-center rounded-lg transition-colors hover:bg-slate-100 hover:text-[#0F172A]"
              title={copied ? 'Скопировано!' : 'Копировать'}
            >
              {copied ? (
                <Check className="h-4 w-4 text-[#04E061]" />
              ) : (
                <Copy className="h-4 w-4" />
              )}
            </button>
            {onRetry && (
              <button
                type="button"
                onClick={() => onRetry(message.id)}
                className="flex h-7 w-7 items-center justify-center rounded-lg transition-colors hover:bg-slate-100 hover:text-[#0F172A]"
                title="Перегенерировать ответ"
              >
                <RotateCcw className="h-4 w-4" />
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
});

export default ChatMessageItem;
