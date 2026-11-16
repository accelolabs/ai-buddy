import { SidebarMenu, SidebarMenuItem, SidebarMenuSkeleton } from '@/shared/ui';

export interface DialogsSkeletonProps {
  count?: number;
}

export function DialogsSkeleton({ count = 4 }: DialogsSkeletonProps) {
  return (
    <SidebarMenu>
      {Array.from({ length: count }).map((_, index) => (
        <SidebarMenuItem key={index}>
          <SidebarMenuSkeleton showIcon />
        </SidebarMenuItem>
      ))}
    </SidebarMenu>
  );
}
