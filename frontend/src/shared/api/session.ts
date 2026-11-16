import { create } from 'zustand';
import { queryClient } from './queryClient';
import { getUserIdCookie, setUserIdCookie, clearUserIdCookie } from '@/entities/user/lib/cookies';

let requests = new AbortController();
export const sessionSignal = () => requests.signal;
export const useSession = create<{ userId: string | null; revision: number }>(() => ({
  userId: getUserIdCookie(),
  revision: 0,
}));

export function changeUser(userId: string | null) {
  requests.abort();
  requests = new AbortController();
  void queryClient.cancelQueries();
  queryClient.clear();
  if (userId) setUserIdCookie(userId);
  else clearUserIdCookie();
  useSession.setState((s) => ({ userId, revision: s.revision + 1 }));
}
