import { useLocation, useNavigate } from 'react-router';
import { BookOpen, MessageSquare, Milestone, SquarePen, Ticket } from 'lucide-react';
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
  SidebarTrigger,
} from '@/shared/ui';
import { UserCard, useUser, isTrainee } from '@/entities/user';
import { useChatStore } from '@/entities/chat';
import chipLogo from '@/assets/chip-logo.svg';

export function AppSidebar() {
  const { data: user } = useUser();
  const navigate = useNavigate();
  const location = useLocation();
  const intern = isTrainee(user);
  const links = [
    ...(intern ? [{ path: '/', label: 'Чат', icon: MessageSquare }] : []),
    { path: '/route', label: 'Маршрут стажера', icon: Milestone },
    { path: '/tickets', label: 'Тикеты', icon: Ticket },
    { path: '/wiki', label: 'База знаний', icon: BookOpen },
  ];
  return (
    <Sidebar collapsible="icon">
      <SidebarHeader className="border-b p-3">
        <div className="flex items-center gap-2">
          <img src={chipLogo} className="h-8 w-8" alt="" />
          <span className="group-data-[collapsible=icon]:hidden font-semibold">AI-buddy</span>
          <SidebarTrigger />
        </div>
      </SidebarHeader>
      <SidebarContent className="p-2">
        <SidebarMenu>
          {intern && (
            <SidebarMenuItem>
              <SidebarMenuButton
                tooltip="Новый чат"
                onClick={() => {
                  useChatStore.getState().startNewChat();
                  navigate('/');
                }}
              >
                <SquarePen />
                <span>Новый чат</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          )}
          {links.map(({ path, label, icon: Icon }) => (
            <SidebarMenuItem key={path}>
              <SidebarMenuButton
                tooltip={label}
                isActive={location.pathname === path}
                onClick={() => navigate(path)}
              >
                <Icon />
                <span>{label}</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          ))}
        </SidebarMenu>
      </SidebarContent>
      <SidebarFooter>
        <UserCard user={user} />
      </SidebarFooter>
    </Sidebar>
  );
}
