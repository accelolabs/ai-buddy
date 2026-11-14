import { SidebarProvider, SidebarInset } from '@/shared/ui';
import { AppSidebar } from '@/widgets/sidebar';
import { Outlet } from 'react-router';

export function AppLayout() {
  return (
    <SidebarProvider className="flex min-h-screen w-full bg-background">
      <AppSidebar />
      <SidebarInset>
        <main className="flex flex-1 flex-col gap-4 p-4 md:p-6">
          <Outlet />
        </main>
      </SidebarInset>
    </SidebarProvider>
  );
}

export default AppLayout;
