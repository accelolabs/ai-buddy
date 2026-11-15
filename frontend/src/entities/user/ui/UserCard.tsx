import { LogOut } from 'lucide-react';
import { useNavigate } from 'react-router';
import { useQueryClient } from '@tanstack/react-query';
import { getUserInitials, getUserRoleLabel } from '@/entities/user/model/helpers';
import { clearUserIdCookie } from '../lib/cookies';
import { useUser } from '../api/useUser';
import type { User } from '../model/types';

export interface UserCardProps {
  user?: User;
  className?: string;
  showRole?: boolean;
  showLogout?: boolean;
  onLogout?: () => void;
}

export function UserCard({
  user: propUser,
  className = '',
  showRole = true,
  showLogout = true,
  onLogout,
}: UserCardProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { data: queryUser, isLoading } = useUser();
  const user = propUser ?? queryUser;

  const handleLogout = async (e: React.MouseEvent) => {
    e.stopPropagation();
    if (onLogout) {
      onLogout();
      return;
    }
    clearUserIdCookie();
    await queryClient.invalidateQueries({ queryKey: ['user'] });
    navigate('/login');
  };

  if (isLoading && !user) {
    return (
      <div className={`flex items-center gap-3 ${className}`}>
        <div className="flex h-8 w-8 shrink-0 animate-pulse items-center justify-center rounded-full bg-primary/10" />
        <div className="flex min-w-0 flex-col gap-1 group-data-[collapsible=icon]:hidden">
          <div className="h-3.5 w-16 animate-pulse rounded bg-sidebar-foreground/20" />
          <div className="h-2.5 w-10 animate-pulse rounded bg-sidebar-foreground/10" />
        </div>
      </div>
    );
  }

  const initial = getUserInitials(user?.name);
  const roleLabel = getUserRoleLabel(user?.role);

  return (
    <div
      className={`flex w-full items-center justify-between gap-2 group-data-[collapsible=icon]:flex-col group-data-[collapsible=icon]:items-center ${className}`}
    >
      <div className="flex min-w-0 items-center gap-3">
        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10 text-sm font-medium text-primary">
          {initial}
        </div>
        <div className="flex min-w-0 flex-col group-data-[collapsible=icon]:hidden">
          <span className="truncate text-sm font-medium text-sidebar-foreground">
            {user?.name ?? 'Пользователь'}
          </span>
          {showRole && (
            <span className="truncate text-xs text-sidebar-foreground/60">
              {roleLabel}
            </span>
          )}
        </div>
      </div>

      {showLogout && (
        <button
          type="button"
          onClick={handleLogout}
          title="Выйти"
          aria-label="Выйти"
          className="flex h-8 shrink-0 items-center gap-1.5 rounded-lg px-2 text-xs font-medium text-sidebar-foreground/70 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-950/30 dark:hover:text-red-400 cursor-pointer"
        >
          <LogOut className="h-4 w-4 shrink-0" />
          <span className="group-data-[collapsible=icon]:hidden">Выйти</span>
        </button>
      )}
    </div>
  );
}

export default UserCard;
