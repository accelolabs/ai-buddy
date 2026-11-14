import { useQuery } from '@tanstack/react-query';
import dialogsData from '../../../../../mock-server/dialogs.json';

export interface Dialog {
  id: string;
  title: string;
  topic?: string;
  createdAt: string;
  updatedAt: string;
}

export function useDialogs() {
  return useQuery<Dialog[]>({
    queryKey: ['dialogs'],
    queryFn: async () => {
      return (dialogsData.dialogs || dialogsData) as Dialog[];
    },
  });
}
