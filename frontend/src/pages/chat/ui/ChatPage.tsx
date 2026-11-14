import { useUser, isMentor } from '@/entities/user';
import { Navigate } from 'react-router';
import { BuddyChat, ChatHeader, ChatBackground } from '@/widgets/ai-buddy-chat';

export function ChatPage() {
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

  return (
    <div className="relative flex min-h-full flex-1 flex-col overflow-hidden bg-white">
      <ChatBackground />
      <ChatHeader />
      <main className="flex flex-1 flex-col">
        <BuddyChat />
      </main>
    </div>
  );
}

export default ChatPage;
