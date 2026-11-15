import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { fetchDialogs, createDialogApi } from './dialogApi';
import { dialogKeys } from './dialogKeys';
import type { Dialog, CreateDialogPayload } from '../model/types';

export function useDialogs() {
  return useQuery<Dialog[]>({
    queryKey: dialogKeys.list(),
    queryFn: fetchDialogs,
    staleTime: 1000 * 60 * 2,
  });
}

export function useCreateDialog() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (payload: CreateDialogPayload) => createDialogApi(payload),
    onSuccess: (newDialog) => {
      queryClient.setQueryData<Dialog[]>(dialogKeys.list(), (old) => {
        return old ? [newDialog, ...old] : [newDialog];
      });
      queryClient.invalidateQueries({ queryKey: dialogKeys.lists() });
    },
  });
}
