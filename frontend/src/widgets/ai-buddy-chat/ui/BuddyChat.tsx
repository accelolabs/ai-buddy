import { useUser, isMentor } from '@/entities/user';
import { AlertCircle } from 'lucide-react';

function BuddyChat() {
  const { data: user, isLoading } = useUser();

  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center p-8 text-muted-foreground">
        Загрузка...
      </div>
    );
  }

  if (isMentor(user)) {
    return (
      <div className="flex flex-1 flex-col items-center justify-center gap-3 rounded-xl border border-destructive/20 bg-destructive/5 p-8 text-center text-destructive">
        <AlertCircle className="h-8 w-8" />
        <h3 className="text-lg font-semibold">Доступ к чату ограничен</h3>
        <p className="max-w-md text-sm text-muted-foreground">
          Чат AI-buddy доступен только стажерам. Для наставников предназначен дашборд «Мои стажеры»
          в Tracker.
        </p>
      </div>
    );
  }

  return <div className="flex flex-1 flex-col rounded-xl border border-border bg-card p-4"></div>;
}

export { BuddyChat };
