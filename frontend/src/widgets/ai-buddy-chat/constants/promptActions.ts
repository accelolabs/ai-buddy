export interface PromptAction {
  title: string;
  subtitle: string;
  promptText: string;
}

export const PROMPT_ACTIONS: PromptAction[] = [
  {
    title: 'Покажи документы команды',
    subtitle: 'ссылка на нужный раздел',
    promptText: 'Покажи документы команды',
  },
  {
    title: 'Защита данных и КИ',
    subtitle: 'ознакомление с инструкциями',
    promptText: 'Защита данных и КИ',
  },
  {
    title: 'Мой прогресс',
    subtitle: 'открытие карты прогресса',
    promptText: '/progress',
  },
  {
    title: 'Покажи памятку',
    subtitle: 'о закрытии практики',
    promptText: 'Покажи памятку о закрытии практики',
  },
];
