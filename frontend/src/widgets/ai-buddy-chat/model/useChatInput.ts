import { useRef, useEffect, type KeyboardEvent, type FocusEvent } from 'react';
import { useChatStore } from '@/entities/chat';

export function useChatInput(onSubmit?: (value: string) => void) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const draft = useChatStore((state) => state.draft);
  const isExpanded = useChatStore((state) => state.isExpanded);
  const setDraft = useChatStore((state) => state.setDraft);
  const setIsExpanded = useChatStore((state) => state.setIsExpanded);
  const resetDraft = useChatStore((state) => state.resetDraft);

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

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      if (draft.trim()) {
        onSubmit?.(draft.trim());
        resetDraft();
      }
    } else if (e.key === 'Escape' && !draft.trim()) {
      setIsExpanded(false);
    }
  };

  const handleSend = () => {
    if (draft.trim()) {
      onSubmit?.(draft.trim());
      resetDraft();
    }
  };

  const handleExpand = () => {
    if (!isExpanded) {
      setIsExpanded(true);
    }
  };

  const handleInsertProgress = () => {
    setDraft('/progress');
    textareaRef.current?.focus();
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
    handleInsertProgress,
  };
}
