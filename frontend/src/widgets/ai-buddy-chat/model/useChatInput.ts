import { useRef, useEffect, type KeyboardEvent, type FocusEvent } from 'react';
import { useChatStore, type ChatStore } from '@/entities/chat';

export function useChatInput(onSubmit?: (value: string) => void) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const busy = useChatStore((s) => s.busy);
  const draft = useChatStore((state: ChatStore) => state.draft);
  const isExpanded = useChatStore((state: ChatStore) => state.isExpanded);
  const setDraft = useChatStore((state: ChatStore) => state.setDraft);
  const setIsExpanded = useChatStore((state: ChatStore) => state.setIsExpanded);
  const resetDraft = useChatStore((state: ChatStore) => state.resetDraft);
  const sendMessage = useChatStore((state: ChatStore) => state.sendMessage);

  useEffect(() => {
    if (isExpanded) {
      textareaRef.current?.focus();
    }
  }, [isExpanded]);

  const handleBlur = (e: FocusEvent<HTMLDivElement>) => {
    if (!e.currentTarget.contains(e.relatedTarget as Node)) {
      if (!draft.trim()) {
        setIsExpanded(false);
      }
    }
  };

  const submitValue = (val: string) => {
    if (busy) return;
    if (onSubmit) {
      onSubmit(val);
    } else {
      sendMessage(val);
    }
    resetDraft();
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      if (draft.trim()) {
        submitValue(draft.trim());
      }
    } else if (e.key === 'Escape' && !draft.trim()) {
      setIsExpanded(false);
    }
  };

  const handleSend = () => {
    if (draft.trim()) {
      submitValue(draft.trim());
    }
  };

  const handleExpand = () => {
    if (!isExpanded) {
      setIsExpanded(true);
    }
  };

  return {
    textareaRef,
    draft,
    isExpanded,
    setDraft,
    handleBlur,
    handleKeyDown,
    handleSend,
    handleExpand,
  };
}
