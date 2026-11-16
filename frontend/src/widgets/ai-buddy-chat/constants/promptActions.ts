export interface PromptAction {
  title: string;
  subtitle: string;
  promptText: string;
}

export const PROMPT_ACTIONS: PromptAction[] = [
  {
    title: 'Какие ценности у моей команды?',
    subtitle: 'принципы работы команды',
    promptText: 'Какие ценности у моей команды?',
  },
  {
    title: 'Покажи мои задачи.',
    subtitle: 'список задач и сроки',
    promptText: 'Покажи мои задачи.',
  },
  {
    title: 'Создай задачу написать балансировщик.',
    subtitle: 'уточним срок и подготовим черновик',
    promptText: 'Создай задачу написать балансировщик.',
  },
  {
    title: 'Создай тикет о неработающем датацентре.',
    subtitle: 'черновик обращения наставнику',
    promptText: 'Создай тикет о неработающем датацентре.',
  },
];
