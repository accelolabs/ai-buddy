import { BuddyChat, ChatBackground } from '@/widgets/ai-buddy-chat';
export function ChatPage() {
  return (
    <div className="relative flex min-h-full flex-1 flex-col overflow-hidden bg-white">
      <ChatBackground />
      <BuddyChat />
    </div>
  );
}
