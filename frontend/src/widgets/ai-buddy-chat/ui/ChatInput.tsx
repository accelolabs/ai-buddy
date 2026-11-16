import { memo } from 'react';
import { Send, Square } from 'lucide-react';
import { useChatStore } from '@/entities/chat';
import { useChatInput } from '../model/useChatInput';

export interface ChatInputProps {
  onSubmit?: (value: string) => void;
  className?: string;
}

export const ChatInput = memo(function ChatInput({ onSubmit, className = '' }: ChatInputProps) {
  const busy = useChatStore((state) => state.busy);
  const stop = useChatStore((state) => state.stop);
  const icon = busy ? (
    <Square className="h-4 w-4 text-white" fill="currentColor" />
  ) : (
    <Send className="h-4 w-4 text-white" />
  );
  const {
    textareaRef,
    draft,
    isExpanded,
    setDraft,
    handleBlur,
    handleKeyDown,
    handleSend,
    handleExpand,
  } = useChatInput(onSubmit);

  return (
    <div
      onBlur={handleBlur}
      onClick={handleExpand}
      className={`relative flex w-full flex-col justify-between bg-white transition-all duration-300 ease-in-out ${
        isExpanded
          ? 'h-[131px] rounded-[20px] border-2 border-[#89C6FC] p-4 shadow-[0_0_15px_rgba(137,198,252,0.35)]'
          : 'h-[54px] cursor-pointer rounded-[20px] border border-slate-200 px-5 py-2 shadow-sm hover:border-slate-300'
      } ${className}`}
    >
      <div className={`flex w-full items-center ${isExpanded ? 'flex-1 items-start' : 'h-full'}`}>
        <textarea
          ref={textareaRef}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={handleKeyDown}
          onFocus={handleExpand}
          placeholder={isExpanded ? 'Спросите что-нибудь у AI-buddy...' : 'Спросите что-нибудь...'}
          rows={isExpanded ? 2 : 1}
          className={`w-full resize-none bg-transparent text-[15px] text-[#0F172A] focus:outline-none ${
            isExpanded
              ? 'leading-relaxed placeholder:text-[#677891]'
              : 'cursor-pointer overflow-hidden py-1.5 placeholder:text-slate-400'
          }`}
        />

        {!isExpanded && (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              if (busy) stop();
              else handleExpand();
            }}
            className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-[#00AAFF] transition-all duration-200 hover:bg-[#0096e6]"
            title={busy ? 'Остановить ответ' : 'Начать ввод'}
          >
            {icon}
          </button>
        )}
      </div>

      {isExpanded && (
        <div className="flex items-center justify-end pt-1">
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              if (busy) stop();
              else handleSend();
            }}
            disabled={!busy && !draft.trim()}
            className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-[#00AAFF] transition-all duration-200 hover:bg-[#0096e6] hover:shadow-md disabled:cursor-not-allowed disabled:opacity-50"
            title={busy ? 'Остановить ответ' : 'Отправить сообщение'}
          >
            {icon}
          </button>
        </div>
      )}
    </div>
  );
});

export default ChatInput;
