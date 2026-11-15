import { useEffect } from 'react';
import { useSearchParams, Navigate } from 'react-router';
import { useUser, isMentor } from '@/entities/user';
import { BuddyChat, ChatBackground } from '@/widgets/ai-buddy-chat';
import { useDialogMessages } from '@/entities/dialog';
import { useChatStore } from '@/entities/chat';

export function ChatPage() {
  const { data: user, isLoading } = useUser();
  const [searchParams, setSearchParams] = useSearchParams();
  const dialogId = searchParams.get('dialogId');

  const { data: loadedMessages, isSuccess } = useDialogMessages(dialogId);
  const loadDialog = useChatStore((s) => s.loadDialog);
  const startNewChat = useChatStore((s) => s.startNewChat);
  const activeDialogId = useChatStore((s) => s.activeDialogId);

  useEffect(() => {
    if (dialogId) {
      if (dialogId !== activeDialogId && isSuccess && loadedMessages) {
        loadDialog(dialogId, loadedMessages);
      }
    } else if (activeDialogId !== null) {
      startNewChat();
    }
  }, [dialogId, activeDialogId, isSuccess, loadedMessages, loadDialog, startNewChat]);

  useEffect(() => {
    if (activeDialogId && !dialogId) {
      setSearchParams({ dialogId: activeDialogId }, { replace: true });
    }
  }, [activeDialogId, dialogId, setSearchParams]);

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
      <main className="flex flex-1 flex-col">
        <BuddyChat />
      </main>
    </div>
  );
}

export default ChatPage;
