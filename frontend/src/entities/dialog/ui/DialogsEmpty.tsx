import { MessageSquareDashed } from 'lucide-react';

export function DialogsEmpty() {
  return (
    <div className="flex flex-col items-center justify-center px-3 py-6 text-center text-xs text-muted-foreground group-data-[collapsible=icon]:hidden">
      <MessageSquareDashed className="mb-2 h-6 w-6 text-muted-foreground/60" />
      <p className="font-medium">Нет сохраненных диалогов</p>
      <p className="mt-1 text-[11px] text-muted-foreground/80">
        Задайте вопрос AI-buddy, чтобы начать диалог
      </p>
    </div>
  );
}
