import type { User } from '../model/types';

export interface MockUserMeta extends User {
  roleLabel: string;
  roleTitle: string;
  infoNote: string;
  mentorName?: string;
  mentees?: string[];
  targetRoute: string;
}

export const MOCK_USERS: MockUserMeta[] = [
  {
    id: 'usr-trainee-dasha',
    name: 'Даша',
    role: 'trainee',
    roleLabel: 'стажер',
    roleTitle: 'Стажёр',
    team: 'Frontend Platform',
    mentorName: 'Катя',
    infoNote: 'Ментор: Катя · Доступ к чату AI-buddy, прогрессу и документам команды.',
    targetRoute: '/',
  },
  {
    id: 'usr-trainee-lucy',
    name: 'Люси',
    role: 'trainee',
    roleLabel: 'стажер',
    roleTitle: 'Стажёр',
    team: 'Frontend Platform',
    mentorName: 'Катя',
    infoNote: 'Ментор: Катя · Доступ к чату AI-buddy, прогрессу и документам команды.',
    targetRoute: '/',
  },
  {
    id: 'usr-mentor-katya',
    name: 'Катя',
    role: 'mentor',
    roleLabel: 'наставник',
    roleTitle: 'Наставник',
    team: 'Frontend Platform',
    mentees: ['Даша', 'Люси'],
    infoNote: 'Доступ к дашборду наставника, прогрессу стажёров и тикетам помощи.',
    targetRoute: '/dashboard',
  },
];

export const DEFAULT_USER = MOCK_USERS[0];
