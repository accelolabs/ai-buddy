export type UserRole = 'trainee' | 'intern' | 'mentor';

export interface User {
  id: string;
  name: string;
  role: UserRole;
  team: string;
  avatarUrl?: string;
}

export function isTrainee(user?: User | null): boolean {
  if (!user) return false;
  return user.role === 'trainee' || user.role === 'intern';
}

export function isMentor(user?: User | null): boolean {
  if (!user) return false;
  return user.role === 'mentor';
}

export function getUserRoleLabel(role?: UserRole): string {
  switch (role) {
    case 'mentor':
      return 'Наставник';
    case 'trainee':
    case 'intern':
    default:
      return 'Стажер';
  }
}

export function getUserInitials(name?: string): string {
  if (!name) return '?';
  const trimmed = name.trim();
  return trimmed.charAt(0).toUpperCase();
}
