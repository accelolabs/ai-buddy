import { MessageSquare, FileText, Code2, Trash2 } from 'lucide-react';
import { SidebarMenuItem, SidebarMenuButton, SidebarMenuAction } from '@/shared/ui';
import type { Dialog } from '../model/types';

export interface DialogMenuItemProps {
  dialog: Dialog;
  isActive?: boolean;
  onSelect?: (dialog: Dialog) => void;
  onDelete?: (dialog: Dialog) => void;
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

export function DialogMenuItem({
  dialog,
  isActive = false,
  onSelect,
  onDelete,
}: DialogMenuItemProps) {
  return (
    <SidebarMenuItem className="group/dialog-item relative">
      <SidebarMenuButton
        isActive={isActive}
        tooltip={dialog.title}
        onClick={() => onSelect?.(dialog)}
        className="cursor-pointer pr-7"
      >
        {getTopicIcon(dialog.topic)}
        <span className="truncate">{dialog.title}</span>
      </SidebarMenuButton>
      {onDelete && (
        <SidebarMenuAction
          showOnHover
          title="Удалить диалог"
          aria-label={`Удалить диалог ${dialog.title}`}
          onClick={(e) => {
            e.stopPropagation();
            onDelete(dialog);
          }}
          className="cursor-pointer text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors"
        >
          <Trash2 className="h-3.5 w-3.5" />
          <span className="sr-only">Удалить</span>
        </SidebarMenuAction>
      )}
    </SidebarMenuItem>
  );
}
