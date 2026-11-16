import type { MentorPayload } from '@/entities/chat';

export const DEFAULT_MENTOR: MentorPayload = {
  id: 'mentor-1',
  name: 'Мария Соколова',
  role: 'Наставник стажёров',
  team: 'Команда онбординга',
  schedule: 'Отвечает в будни с 10:00 до 19:00',
  initials: 'МС',
  isOnline: true,
};

export const MENTOR_SCHEDULE_NOTICE = 'Среднее время первого ответа — 15 минут.';
