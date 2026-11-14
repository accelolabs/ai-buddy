export type StageStatus = 'pending' | 'in_progress' | 'completed' | 'error';

export interface MessageStage {
  id: string;
  label: string;
  status: StageStatus;
}

export interface WikiSourcePayload {
  title: string;
  section: string;
  quote: string;
  url?: string;
}

export interface MentorPayload {
  id: string;
  name: string;
  role: string;
  team: string;
  schedule: string;
  initials: string;
  isOnline?: boolean;
}

export type MessageCard =
  | { type: 'wiki_source'; payload: WikiSourcePayload }
  | { type: 'mentor_fallback'; payload: MentorPayload }
  | { type: 'custom'; payload: unknown };

export type MessageStatus = 'in_progress' | 'completed' | 'error';

export interface ChatMessage {
  id: string;
  sender: 'user' | 'buddy';
  text: string;
  status: MessageStatus;
  stages?: MessageStage[];
  cards?: MessageCard[];
  createdAt: string;
  lastSeq?: number;
  error?: string;
}

export type SSEEvent =
  | { type: 'stage'; seq: number; id: string; label: string; status: StageStatus }
  | { type: 'delta'; seq: number; delta: string; offset?: number }
  | {
      type: 'card';
      seq: number;
      cardType: 'wiki_source' | 'mentor_fallback';
      payload: WikiSourcePayload | MentorPayload;
    }
  | { type: 'done'; seq: number; finalText?: string }
  | { type: 'error'; seq: number; error: string; canRetry?: boolean };
