import { useUser } from '@/entities/user';
import { PromptButton } from '@/shared/ui';
import { BuddyChat } from '@/widgets/ai-buddy-chat';

export function ChatPage() {
  const { data: user } = useUser();

  const teamName = user?.team ?? 'команды';

  return (
    <div className="flex flex-1 flex-col gap-6">
      <div className="flex flex-col gap-2">
        <h1 className="text-2xl font-bold tracking-tight">Привет, {user?.name ?? 'стажер'}!</h1>
        <p className="text-muted-foreground">
          Я твой AI-buddy. Помогу освоиться в компании, найти нужные документы и отслеживать
          прогресс адаптации.
        </p>
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <PromptButton title="Документы команды" subtitle={`База знаний «${teamName}»`} />
        <PromptButton title="Защита данных" subtitle="Регламенты и безопасность" />
        <PromptButton title="Мой прогресс" subtitle="План адаптации и задачи" />
        <PromptButton title="Памятка о практике" subtitle="Полезная информация" />
      </div>

      <BuddyChat />
    </div>
  );
}

export default ChatPage;
