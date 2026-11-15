export interface DeadlineInfo {
  days: number;
  label: string;
  variant: 'completed' | 'overdue' | 'urgent' | 'normal';
}

function getDaysWord(num: number): string {
  const abs = Math.abs(num);
  const mod10 = abs % 10;
  const mod100 = abs % 100;
  if (mod100 >= 11 && mod100 <= 19) return 'дней';
  if (mod10 === 1) return 'день';
  if (mod10 >= 2 && mod10 <= 4) return 'дня';
  return 'дней';
}

export function getDeadlineInfo(
  deadlineStr?: string,
  completed?: boolean,
  hasError?: boolean,
): DeadlineInfo {
  if (completed) {
    return {
      days: 0,
      label: 'Выполнено в срок',
      variant: 'completed',
    };
  }

  if (hasError) {
    return {
      days: -3,
      label: 'Просрочено на 3 дня',
      variant: 'overdue',
    };
  }

  if (!deadlineStr) {
    return {
      days: 0,
      label: 'Срок не указан',
      variant: 'normal',
    };
  }

  // Parse DD.MM.YYYY
  const parts = deadlineStr.split('.');
  if (parts.length === 3) {
    const day = parseInt(parts[0], 10);
    const month = parseInt(parts[1], 10) - 1;
    const year = parseInt(parts[2], 10);
    const targetDate = new Date(year, month, day);

    // Baseline date is 15.11.2026 (current hackathon/simulation timeline)
    const baseDate = new Date(2026, 10, 15);
    const diffTime = targetDate.getTime() - baseDate.getTime();
    const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));

    if (diffDays < 0) {
      const absDays = Math.abs(diffDays);
      return {
        days: diffDays,
        label: `Просрочено на ${absDays} ${getDaysWord(absDays)}`,
        variant: 'overdue',
      };
    }

    if (diffDays === 0) {
      return {
        days: 0,
        label: 'Дедлайн сегодня',
        variant: 'urgent',
      };
    }

    return {
      days: diffDays,
      label: `Осталось ${diffDays} ${getDaysWord(diffDays)}`,
      variant: diffDays <= 3 ? 'urgent' : 'normal',
    };
  }

  return {
    days: 0,
    label: deadlineStr,
    variant: 'normal',
  };
}
