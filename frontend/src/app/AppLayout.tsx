import { Navigate, Outlet } from 'react-router';
import { SidebarProvider, SidebarInset, SidebarTrigger } from '@/shared/ui';
import { RequestState } from '@/shared/ui/RequestState';
import { AppSidebar } from '@/widgets/sidebar';
import { useUser } from '@/entities/user';
import { useSession } from '@/shared/api/session';

export function AppLayout() {
  const user = useUser();
  const { userId, revision } = useSession();
  if (!userId) return <Navigate to="/login" replace />;
  if (user.isPending || user.error)
    return (
      <RequestState loading={user.isPending} error={user.error} retry={() => void user.refetch()} />
    );
  if (!user.data) return <Navigate to="/login" replace />;
  return (
    <SidebarProvider key={revision} className="flex min-h-screen w-full bg-background">
      <AppSidebar />
      <SidebarInset>
        <main className="flex flex-1 flex-col gap-4 p-4 md:p-6">
          <SidebarTrigger className="md:hidden" />
          <Outlet />
        </main>
      </SidebarInset>
    </SidebarProvider>
  );
}
