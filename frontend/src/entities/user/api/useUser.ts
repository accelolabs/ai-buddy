import { useQuery } from '@tanstack/react-query';
import type { User } from '../model/types';
import userData from '../../../../../mock-server/user.json';

export function useUser() {
  return useQuery<User>({
    queryKey: ['user'],
    queryFn: async () => {
      const data = userData as { user?: User } | User;
      if ('user' in data && data.user) {
        return data.user;
      }
      return data as User;
    },
  });
}
