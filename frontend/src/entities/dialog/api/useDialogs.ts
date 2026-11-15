import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  fetchDialogs,
  createDialogApi,
  deleteDialogApi,
  fetchDialogMessages,
} from './dialogApi';
import { dialogKeys } from './dialogKeys';
import type { Dialog, CreateDialogPayload } from '../model/types';
import type { ChatMessage } from '@/entities/chat/model/types';

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

export function useDeleteDialog() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (dialogId: string) => deleteDialogApi(dialogId),
    onSuccess: (_, deletedId) => {
      queryClient.setQueryData<Dialog[]>(dialogKeys.list(), (old) => {
        return old ? old.filter((item) => item.id !== deletedId) : [];
      });
      queryClient.removeQueries({ queryKey: dialogKeys.detail(deletedId) });
      queryClient.invalidateQueries({ queryKey: dialogKeys.lists() });
    },
  });
}

export function useDialogMessages(dialogId: string | null) {
  return useQuery<ChatMessage[]>({
    queryKey: dialogId ? dialogKeys.messages(dialogId) : ['dialogs', 'none', 'messages'],
    queryFn: () => (dialogId ? fetchDialogMessages(dialogId) : Promise.resolve([])),
    enabled: Boolean(dialogId),
    staleTime: 1000 * 60 * 5,
  });
}
