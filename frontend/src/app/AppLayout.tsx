import { Navigate, NavLink, Outlet, useLocation } from 'react-router';
import { RequestState } from '@/shared/ui/RequestState';
import { UserCard, useUser, isTrainee } from '@/entities/user';
import { useSession } from '@/shared/api/session';

export function AppLayout() {
  const user = useUser();
  const { pathname } = useLocation();
  const { userId, revision } = useSession();
  if (!userId) return <Navigate to="/login" replace />;
  if (user.isPending || user.error)
    return (
      <RequestState loading={user.isPending} error={user.error} retry={() => void user.refetch()} />
    );
  if (!user.data) return <Navigate to="/login" replace />;

  const links = [
    ...(isTrainee(user.data) ? [{ path: '/', label: 'Чат' }] : []),
    { path: '/route', label: 'Задачи' },
    { path: '/tickets', label: 'Тикеты' },
    { path: '/wiki', label: 'Знания' },
  ];

  return (
    <div key={revision} className="flex min-h-dvh min-w-0 flex-col bg-white">
      <header className="relative h-20 shrink-0">
        <nav
          aria-label="Приложения"
          className="absolute left-1/2 top-4 max-w-[calc(100%_-_8rem)] -translate-x-1/2 overflow-x-auto rounded-xl bg-[#F1F5F9] p-1"
        >
          <div className="flex w-max gap-1">
            {links.map(({ path, label }) => (
              <NavLink
                key={path}
                to={path}
                end={path === '/'}
                className={({ isActive }) =>
                  `shrink-0 rounded-lg px-5 py-2 text-sm font-medium transition-colors focus-visible:outline-2 focus-visible:outline-sky-500 ${
                    isActive ||
                    (path === '/' && pathname === '/chat') ||
                    (path === '/route' && pathname === '/dashboard')
                      ? 'bg-white text-[#0F172A] shadow-xs'
                      : 'text-[#64748B] hover:text-[#0F172A]'
                  }`
                }
              >
                {label}
              </NavLink>
            ))}
          </div>
        </nav>
        <div className="absolute right-4 top-4 md:right-6">
          <UserCard user={user.data} />
        </div>
      </header>
      <main className="flex min-w-0 flex-1 flex-col">
        <Outlet />
      </main>
    </div>
  );
}
