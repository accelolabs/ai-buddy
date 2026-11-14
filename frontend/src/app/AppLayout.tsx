import { SidebarProvider, SidebarInset, SidebarTrigger } from '@/shared/ui/sidebar';
import { AppSidebar } from '@/widgets/sidebar';
import { Outlet } from 'react-router';

export function AppLayout() {
  return (
    <SidebarProvider className="flex min-h-screen w-full bg-background">
      <AppSidebar />
      <SidebarInset>
        <header className="flex h-14 shrink-0 items-center gap-2 border-b px-4">
          <SidebarTrigger className="-ml-1" />
        </header>
        <main className="flex flex-1 flex-col gap-4 p-4 md:p-6">
          <Outlet />
        </main>
      </SidebarInset>
    </SidebarProvider>
  );
}

export default AppLayout;
