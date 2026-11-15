export const dialogKeys = {
  all: ['dialogs'] as const,
  lists: () => [...dialogKeys.all, 'list'] as const,
  list: (filters?: Record<string, unknown>) => [...dialogKeys.lists(), { filters }] as const,
  details: () => [...dialogKeys.all, 'detail'] as const,
  detail: (id: string) => [...dialogKeys.details(), id] as const,
};
