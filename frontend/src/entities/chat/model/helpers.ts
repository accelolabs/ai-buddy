import type {
  ChatMessage,
  SSEEvent,
  MessageStage,
  WikiSourcePayload,
  MentorPayload,
} from './types';

export function createUserMessage(text: string): ChatMessage {
  return {
    id: `user-${Date.now()}`,
    sender: 'user',
    text,
    status: 'completed',
    createdAt: new Date().toISOString(),
  };
}

export function createBuddyMessage(): ChatMessage {
  return {
    id: `buddy-${Date.now() + 1}`,
    sender: 'buddy',
    text: '',
    status: 'in_progress',
    stages: [],
    cards: [],
    createdAt: new Date().toISOString(),
    lastSeq: 0,
  };
}

function updateStages(
  stages: MessageStage[] = [],
  event: Extract<SSEEvent, { type: 'stage' }>,
): MessageStage[] {
  const nextStages = [...stages];
  const index = nextStages.findIndex((s) => s.id === event.id);
  const stageItem: MessageStage = {
    id: event.id,
    label: event.label,
    status: event.status,
  };

  if (index >= 0) {
    nextStages[index] = stageItem;
  } else {
    nextStages.push(stageItem);
  }

  return nextStages;
}

export function applySSEEventToMessage(message: ChatMessage, event: SSEEvent): ChatMessage {
  if (message.lastSeq !== undefined && event.seq <= message.lastSeq) {
    return message;
  }

  const updated: ChatMessage = {
    ...message,
    lastSeq: event.seq,
  };

  switch (event.type) {
    case 'stage':
      return {
        ...updated,
        stages: updateStages(updated.stages, event),
      };
    case 'delta':
      return {
        ...updated,
        text: (updated.text || '') + event.delta,
      };
    case 'card': {
      const cards = [...(updated.cards || [])];
      if (event.cardType === 'wiki_source') {
        cards.push({
          type: 'wiki_source',
          payload: event.payload as WikiSourcePayload,
        });
      } else if (event.cardType === 'mentor_fallback') {
        cards.push({
          type: 'mentor_fallback',
          payload: event.payload as MentorPayload,
        });
      }
      return {
        ...updated,
        cards,
      };
    }
    case 'done':
      return {
        ...updated,
        status: 'completed',
        text: event.finalText ?? updated.text,
      };
    case 'error':
      return {
        ...updated,
        status: 'error',
        error: event.error,
      };
    default:
      return updated;
  }
}

export function updateMessagesWithEvent(
  messages: ChatMessage[],
  buddyMessageId: string,
  event: SSEEvent,
): ChatMessage[] {
  return messages.map((message) =>
    message.id === buddyMessageId ? applySSEEventToMessage(message, event) : message,
  );
}
