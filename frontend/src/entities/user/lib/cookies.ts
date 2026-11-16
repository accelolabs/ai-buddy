export const USER_ID_COOKIE = 'user_id';
const DEFAULT_MAX_AGE = 60 * 60 * 24 * 30;

export function getUserIdCookie(): string | null {
  if (typeof document === 'undefined') return null;
  const match = document.cookie.match(new RegExp(`(?:^|; )${USER_ID_COOKIE}=([^;]*)`));
  return match ? decodeURIComponent(match[1]) : null;
}

export function setUserIdCookie(userId: string, maxAge = DEFAULT_MAX_AGE): void {
  if (typeof document === 'undefined') return;
  document.cookie = `${USER_ID_COOKIE}=${encodeURIComponent(userId)}; path=/; max-age=${maxAge}; SameSite=Lax`;
}

export function clearUserIdCookie(): void {
  if (typeof document === 'undefined') return;
  document.cookie = `${USER_ID_COOKIE}=; path=/; max-age=0; SameSite=Lax`;
}
