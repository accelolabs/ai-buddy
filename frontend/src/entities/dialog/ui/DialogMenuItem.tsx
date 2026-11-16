import { MessageSquare, FileText, Code2 } from 'lucide-react';
import { SidebarMenuItem, SidebarMenuButton } from '@/shared/ui';
import type { Dialog } from '../model/types';

export interface DialogMenuItemProps {
  dialog: Dialog;
  isActive?: boolean;
  onSelect?: (dialog: Dialog) => void;
}

function getTopicIcon(topic?: string) {
  switch (topic) {
    case 'docs':
      return <FileText className="h-4 w-4 shrink-0 text-muted-foreground" />;
    case 'code':
      return <Code2 className="h-4 w-4 shrink-0 text-muted-foreground" />;
    case 'chat':
    default:
      return <MessageSquare className="h-4 w-4 shrink-0 text-muted-foreground" />;
  }
}

export function DialogMenuItem({ dialog, isActive = false, onSelect }: DialogMenuItemProps) {
  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        isActive={isActive}
        tooltip={dialog.title}
        onClick={() => onSelect?.(dialog)}
        className="cursor-pointer"
      >
        {getTopicIcon(dialog.topic)}
        <span className="truncate">{dialog.title}</span>
      </SidebarMenuButton>
    </SidebarMenuItem>
  );
}
