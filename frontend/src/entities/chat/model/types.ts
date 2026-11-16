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

export interface TicketDraftPayload {
  id?: string;
  subject: string;
  message: string;
  mentor: MentorPayload;
  status: 'draft' | 'submitting' | 'submitted' | 'cancelled';
  ticketId?: string;
  ticketUrl?: string;
}

export interface CreatedTicketPayload {
  id: string;
  subject: string;
  message: string;
  mentorId: string;
  url: string;
  createdAt: string;
}

export type TicketFlowStep = 'idle' | 'awaiting_problem' | 'draft_ready' | 'submitted';

export type MessageCard =
  | { type: 'wiki_source'; payload: WikiSourcePayload }
  | { type: 'mentor_fallback'; payload: MentorPayload }
  | { type: 'ticket_draft'; payload: TicketDraftPayload }
  | { type: 'ticket_success'; payload: CreatedTicketPayload }
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
      cardType: 'wiki_source' | 'mentor_fallback' | 'ticket_draft' | 'ticket_success';
      payload: WikiSourcePayload | MentorPayload | TicketDraftPayload | CreatedTicketPayload;
    }
  | { type: 'flow_step'; seq: number; step: TicketFlowStep; draft?: TicketDraftPayload }
  | { type: 'done'; seq: number; finalText?: string }
  | { type: 'error'; seq: number; error: string; canRetry?: boolean };
