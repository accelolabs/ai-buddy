import type { UserRole } from '@/entities/user/model/types';

export function getUserRoleLabel(role?: UserRole): string {
  switch (role) {
    case 'mentor':
      return 'Наставник';
    case 'trainee':
    default:
      return 'Стажер';
  }
}

export function getUserInitials(name?: string): string {
  if (!name) return '?';
  const trimmed = name.trim();
  return trimmed.charAt(0).toUpperCase();
}
