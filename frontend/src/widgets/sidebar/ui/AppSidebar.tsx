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
  SidebarTrigger,
  useSidebar,
} from '@/shared/ui';
import { SquarePen, MessageSquare } from 'lucide-react';
import { useDialogs } from '../api/useDialogs';
import { Picture } from '@/shared/ui';
import chipLogo from '@/assets/chip-logo.svg';

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const { data: dialogs = [] } = useDialogs();
  const { setOpen } = useSidebar();

  const handleNewChat = () => {};

  return (
    <Sidebar
      collapsible="icon"
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
      {...props}
    >
      <SidebarHeader className="border-b border-sidebar-border px-3 py-3">
        <div className="flex items-center justify-between group-data-[collapsible=icon]:flex-col group-data-[collapsible=icon]:gap-2">
          <div className="flex items-center gap-2 overflow-hidden group-data-[collapsible=icon]:hidden">
            <div className="flex h-8 w-8 shrink-0 items-center justify-center">
              <Picture src={chipLogo} alt="chip logo" priority={true} />
            </div>
            <span className="truncate font-semibold text-sidebar-foreground">AI-buddy</span>
          </div>
          <div className="flex items-center gap-1 group-data-[collapsible=icon]:flex-col group-data-[collapsible=icon]:gap-2">
            <SidebarTrigger className="hidden group-data-[collapsible=icon]:inline-flex h-8 w-8 text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground" />
            <button
              type="button"
              onClick={handleNewChat}
              className="flex h-8 w-8 items-center justify-center rounded-lg text-sidebar-foreground/70 transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground"
              title="Новый чат"
            >
              <SquarePen className="h-4 w-4" />
            </button>
          </div>
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
