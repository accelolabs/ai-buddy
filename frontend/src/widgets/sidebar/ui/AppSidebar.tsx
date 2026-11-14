import * as React from 'react';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
} from '@/shared/ui/sidebar';
import { Bot, SquarePen, MessageSquare } from 'lucide-react';
import { useDialogs } from '../api/useDialogs';

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const { data: dialogs = [] } = useDialogs();

  const handleNewChat = () => {};

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader className="border-b border-sidebar-border px-3 py-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2 overflow-hidden">
            <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
              <Bot className="h-4 w-4" />
            </div>
            <span className="truncate font-semibold text-sidebar-foreground">AI-buddy</span>
          </div>
          <button
            type="button"
            onClick={handleNewChat}
            className="flex h-8 w-8 items-center justify-center rounded-lg text-sidebar-foreground/70 transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
            title="Новый чат"
          >
            <SquarePen className="h-4 w-4" />
          </button>
        </div>
      </SidebarHeader>

      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>История диалогов</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {dialogs.map((dialog) => (
                <SidebarMenuItem key={dialog.id}>
                  <SidebarMenuButton tooltip={dialog.title}>
                    <MessageSquare className="h-4 w-4 shrink-0" />
                    <span className="truncate">{dialog.title}</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter className="border-t border-sidebar-border p-3">
        <div className="flex items-center gap-3">
          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10 text-sm font-medium text-primary">
            Л
          </div>
          <div className="flex min-w-0 flex-col">
            <span className="truncate text-sm font-medium text-sidebar-foreground">Люси</span>
            <span className="truncate text-xs text-sidebar-foreground/60">Стажер</span>
          </div>
        </div>
      </SidebarFooter>
    </Sidebar>
  );
}

export default AppSidebar;
