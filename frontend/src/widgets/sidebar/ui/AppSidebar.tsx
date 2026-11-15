import * as React from 'react';
import { useNavigate, useSearchParams, useLocation } from 'react-router';
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
  Picture,
  Dialog as ModalDialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
  Button,
} from '@/shared/ui';
import { SquarePen, RotateCw, Ticket as TicketIcon, Milestone } from 'lucide-react';
import chipLogo from '@/assets/chip-logo.svg';
import { UserCard, useUser, isTrainee } from '@/entities/user';
import {
  useDialogs,
  useDeleteDialog,
  DialogMenuItem,
  DialogsSkeleton,
  DialogsEmpty,
  type Dialog,
} from '@/entities/dialog';
import { useChatStore } from '@/entities/chat';

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const { data: user } = useUser();
  const isUserTrainee = isTrainee(user);

  const {
    data: dialogs = [],
    isLoading,
    isError,
    refetch,
    isFetching,
  } = useDialogs();

  const deleteDialogMutation = useDeleteDialog();
  const [dialogToDelete, setDialogToDelete] = React.useState<Dialog | null>(null);

  const { setOpen } = useSidebar();
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams, setSearchParams] = useSearchParams();

  const activeDialogId = useChatStore((s) => s.activeDialogId);
  const setActiveDialogId = useChatStore((s) => s.setActiveDialogId);
  const startNewChat = useChatStore((s) => s.startNewChat);

  const currentDialogId = searchParams.get('dialogId') || activeDialogId;

  const handleSelectDialog = (dialog: Dialog) => {
    setActiveDialogId(dialog.id);
    setSearchParams({ dialogId: dialog.id });
    navigate(`/?dialogId=${dialog.id}`);
  };

  const handleNewChat = () => {
    startNewChat();
    setSearchParams({}, { replace: true });
    navigate('/', { replace: true });
  };

  const handleDeleteConfirm = async () => {
    if (!dialogToDelete) return;
    const targetId = dialogToDelete.id;
    try {
      await deleteDialogMutation.mutateAsync(targetId);
      if (currentDialogId === targetId) {
        handleNewChat();
      }
    } catch (error) {
      console.error('Failed to delete dialog:', error);
    } finally {
      setDialogToDelete(null);
    }
  };

  return (
    <>
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
              {isUserTrainee && (
                <button
                  type="button"
                  onClick={handleNewChat}
                  className="flex h-8 w-8 items-center justify-center rounded-lg text-sidebar-foreground/70 transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground cursor-pointer"
                  title="Новый чат"
                >
                  <SquarePen className="h-4 w-4" />
                </button>
              )}
            </div>
          </div>
        </SidebarHeader>

        <SidebarContent>
          <SidebarGroup>
            <SidebarGroupContent>
              <SidebarMenu>
                <SidebarMenuItem>
                  <SidebarMenuButton
                    tooltip="Маршрут стажера"
                    isActive={location.pathname === '/route' || location.pathname === '/dashboard'}
                    onClick={() => navigate('/route')}
                  >
                    <Milestone className="h-4 w-4 shrink-0 text-cyan-600" />
                    <span>Маршрут стажера</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
                <SidebarMenuItem>
                  <SidebarMenuButton
                    tooltip="Обращения и тикеты"
                    onClick={() => navigate('/tickets')}
                  >
                    <TicketIcon className="h-4 w-4 shrink-0" />
                    <span>Тикеты</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>

          {isUserTrainee && (
            <SidebarGroup>
              <SidebarGroupLabel className="flex items-center justify-between">
                <span>История диалогов</span>
                {isFetching && !isLoading && (
                  <RotateCw className="h-3 w-3 animate-spin text-muted-foreground" />
                )}
              </SidebarGroupLabel>
              <SidebarGroupContent>
                {isLoading && <DialogsSkeleton count={4} />}

                {isError && (
                  <div className="flex flex-col items-center justify-center gap-2 p-3 text-center text-xs text-destructive group-data-[collapsible=icon]:hidden">
                    <p>Не удалось загрузить историю диалогов</p>
                    <button
                      type="button"
                      onClick={() => refetch()}
                      className="inline-flex items-center gap-1 rounded bg-secondary px-2 py-1 text-[11px] font-medium text-foreground hover:bg-secondary/80 cursor-pointer"
                    >
                      <RotateCw className="h-3 w-3" />
                      Повторить
                    </button>
                  </div>
                )}

                {!isLoading && !isError && dialogs.length === 0 && <DialogsEmpty />}

                {!isLoading && !isError && dialogs.length > 0 && (
                  <SidebarMenu>
                    {dialogs.map((dialog) => (
                      <DialogMenuItem
                        key={dialog.id}
                        dialog={dialog}
                        isActive={currentDialogId === dialog.id}
                        onSelect={handleSelectDialog}
                        onDelete={setDialogToDelete}
                      />
                    ))}
                  </SidebarMenu>
                )}
              </SidebarGroupContent>
            </SidebarGroup>
          )}
        </SidebarContent>

        <SidebarFooter className="border-t border-sidebar-border p-3">
          <UserCard user={user} />
        </SidebarFooter>
      </Sidebar>

      <ModalDialog
        open={Boolean(dialogToDelete)}
        onOpenChange={(open) => {
          if (!open) setDialogToDelete(null);
        }}
      >
        <DialogContent showCloseButton className="sm:max-w-[420px]">
          <DialogHeader>
            <DialogTitle>Удалить диалог?</DialogTitle>
            <DialogDescription className="mt-2 text-sm text-slate-500">
              Вы уверены, что хотите удалить диалог{' '}
              <span className="font-semibold text-slate-800">
                «{dialogToDelete?.title}»
              </span>
              ? История сообщений будет удалена без возможности восстановления.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="mt-6 flex items-center justify-end gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setDialogToDelete(null)}
              disabled={deleteDialogMutation.isPending}
            >
              Отмена
            </Button>
            <Button
              variant="destructive"
              size="sm"
              onClick={handleDeleteConfirm}
              disabled={deleteDialogMutation.isPending}
            >
              {deleteDialogMutation.isPending ? 'Удаление...' : 'Удалить'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </ModalDialog>
    </>
  );
}

export default AppSidebar;
