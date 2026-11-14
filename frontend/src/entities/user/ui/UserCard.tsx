import { getUserInitials, getUserRoleLabel } from '@/entities/user/model/helpers';
import { useUser } from '../api/useUser';
import type { User } from '../model/types';

export interface UserCardProps {
  user?: User;
  className?: string;
  showRole?: boolean;
}

export function UserCard({ user: propUser, className = '', showRole = true }: UserCardProps) {
  const { data: queryUser, isLoading } = useUser();
  const user = propUser ?? queryUser;

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
    <div className={`flex items-center gap-3 ${className}`}>
      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10 text-sm font-medium text-primary">
        {initial}
      </div>
      <div className="flex min-w-0 flex-col group-data-[collapsible=icon]:hidden">
        <span className="truncate text-sm font-medium text-sidebar-foreground">
          {user?.name ?? 'Пользователь'}
        </span>
        {showRole && (
          <span className="truncate text-xs text-sidebar-foreground/60">{roleLabel}</span>
        )}
      </div>
    </div>
  );
}
