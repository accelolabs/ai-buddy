import { useUser, isMentor } from '@/entities/user';
import { Navigate } from 'react-router';

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
    return <Navigate to="/dashboard" replace />;
  }

  return <div className="flex flex-1 flex-col rounded-xl border border-border bg-card p-4"></div>;
}

export { BuddyChat };
