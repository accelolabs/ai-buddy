import { useQuery } from '@tanstack/react-query';
import type { User } from '../model/types';
import { MOCK_USERS, DEFAULT_USER } from '../constants/mockUsers';
import { getUserIdCookie } from '../lib/cookies';

export function useUser() {
  return useQuery<User>({
    queryKey: ['user'],
    queryFn: async () => {
      const cookieUserId = getUserIdCookie();
      if (!cookieUserId) {
        return DEFAULT_USER;
      }
      const found = MOCK_USERS.find((u) => u.id === cookieUserId);
      return found || DEFAULT_USER;
    },
  });
}
