import { PromptButton, Picture } from '@/shared/ui';
import chipLogo from '@/assets/chip-logo.svg';
import { useChatStore } from '@/entities/chat';
import { ChatInput } from './ChatInput';

const PROMPT_ACTIONS = [
  {
    title: 'Покажи документы команды',
    subtitle: 'ссылка на нужный раздел',
    promptText: 'Покажи документы команды',
  },
  {
    title: 'Защита данных и КИ',
    subtitle: 'ознакомление с инструкциями',
    promptText: 'Защита данных и КИ',
  },
  {
    title: 'Мой прогресс',
    subtitle: 'открытие карты прогресса',
    promptText: '/progress',
  },
  {
    title: 'Покажи памятку',
    subtitle: 'о закрытии практики',
    promptText: 'Покажи памятку о закрытии практики',
  },
];

export function BuddyChat() {
  const applyPrompt = useChatStore((state) => state.applyPrompt);

  return (
    <div className="relative z-10 flex flex-1 flex-col items-center justify-center px-4 py-8">
      <div className="flex w-full max-w-[712px] flex-col items-center gap-8">
        <div className="flex h-14 w-14 items-center justify-center drop-shadow-[0_0_12px_rgba(0,170,255,0.25)]">
          <Picture src={chipLogo} alt="AI-buddy logo" priority={true} className="h-full w-full" />
        </div>

        <div className="flex flex-col items-center gap-2 text-center">
          <h1 className="text-3xl font-extrabold tracking-tight text-[#0F172A] sm:text-4xl">
            Привет! Я — твой AI-buddy.
          </h1>
          <p className="text-xl font-medium text-[#64748B]">Чем помочь сегодня?</p>
        </div>

        <div className="grid w-full grid-cols-1 gap-4 sm:grid-cols-2">
          {PROMPT_ACTIONS.map((action) => (
            <PromptButton
              key={action.title}
              title={action.title}
              subtitle={action.subtitle}
              onClick={() => applyPrompt(action.promptText)}
            />
          ))}
        </div>

        <ChatInput />
      </div>
    </div>
  );
}

export default BuddyChat;
