import type { ReactNode } from 'react';
import { Navigate } from 'react-router';
import { useUser, isMentor } from '@/entities/user';

interface TraineeRouteProps {
  children: ReactNode;
}

export function TraineeRoute({ children }: TraineeRouteProps) {
  const { data: user, isLoading } = useUser();

  if (isLoading) {
    return (
      <div className="flex flex-1 items-center justify-center p-8 text-muted-foreground">
        Загрузка...
      </div>
    );
  }

  if (isMentor(user)) {
    return <Navigate to="/dashboard" replace />;
  }

  return <>{children}</>;
}
