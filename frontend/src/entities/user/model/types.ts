export type UserRole = 'trainee' | 'mentor';

export interface User {
  id: string;
  name: string;
  role: UserRole;
  team: string;
  avatarUrl?: string;
}

export function isTrainee(user?: User | null): boolean {
  if (!user) return false;
  return user.role === 'trainee';
}

export function isMentor(user?: User | null): boolean {
  if (!user) return false;
  return user.role === 'mentor';
}
