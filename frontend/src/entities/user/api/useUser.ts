import { useQuery } from '@tanstack/react-query';
import { request } from '@/shared/api';
import { useSession } from '@/shared/api/session';
import type { User } from '../model/types';

export function useUsers() {
  const revision = useSession((s) => s.revision);
  return useQuery({
    queryKey: ['users', revision],
    queryFn: ({ signal }) => request<User[]>('/api/users', { signal }),
  });
}

export function useUser() {
  const userId = useSession((s) => s.userId);
  const result = useUsers();
  return { ...result, data: result.data?.find((user) => user.id === userId) };
}
