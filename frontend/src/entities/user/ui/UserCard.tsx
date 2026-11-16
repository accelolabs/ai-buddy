import { Menu } from '@base-ui/react/menu';
import { LogOut } from 'lucide-react';
import { useNavigate } from 'react-router';
import { changeUser } from '@/shared/api/session';
import { getUserInitials, getUserRoleLabel } from '@/entities/user/model/helpers';
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
  const { data: queryUser, isLoading } = useUser();
  const user = propUser ?? queryUser;

  const handleLogout = () => {
    if (onLogout) {
      onLogout();
      return;
    }
    changeUser(null);
    navigate('/login');
  };

  return (
    <Menu.Root>
      <Menu.Trigger
        disabled={isLoading && !user}
        aria-label={`Профиль: ${user?.name ?? 'Пользователь'}`}
        className={`flex h-11 w-11 shrink-0 cursor-pointer items-center justify-center rounded-full bg-slate-100 text-sm font-semibold text-slate-700 transition-colors hover:bg-slate-200 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-500 disabled:animate-pulse ${className}`}
      >
        {getUserInitials(user?.name)}
      </Menu.Trigger>
      <Menu.Portal>
        <Menu.Positioner align="end" sideOffset={8} collisionPadding={16} className="z-50">
          <Menu.Popup className="w-64 max-w-[calc(100vw-2rem)] rounded-2xl border border-slate-200 bg-white p-1.5 shadow-lg outline-none">
            <div className="px-3 py-2.5">
              <p className="break-words text-sm font-semibold text-slate-900">
                {user?.name ?? 'Пользователь'}
              </p>
              {showRole && (
                <p className="mt-1 text-xs text-slate-500">{getUserRoleLabel(user?.role)}</p>
              )}
            </div>
            {showLogout && (
              <>
                <Menu.Separator className="mx-2 my-1 h-px bg-slate-100" />
                <Menu.Item
                  onClick={handleLogout}
                  className="flex cursor-pointer items-center gap-2 rounded-lg px-3 py-2.5 text-sm text-slate-600 outline-none data-highlighted:bg-red-50 data-highlighted:text-red-600"
                >
                  <LogOut className="h-4 w-4" />
                  Выйти
                </Menu.Item>
              </>
            )}
          </Menu.Popup>
        </Menu.Positioner>
      </Menu.Portal>
    </Menu.Root>
  );
}

export default UserCard;
