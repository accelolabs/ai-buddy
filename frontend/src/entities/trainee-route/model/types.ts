export type RouteTaskStatus = 'completed' | 'error' | 'pending' | 'in_progress';

export interface RouteTask {
  id: string;
  order: number;
  title: string;
  deadline: string;
  subtitle: string;
  status: RouteTaskStatus;
  completed: boolean;
  hasError?: boolean;
  errorMessage?: string;
  isGlitching?: boolean;
}

export interface TraineeIntern {
  id: string;
  name: string;
  role: string;
  mentor: string;
  startDate: string;
  goal: string;
  expectations: string;
  tasks: RouteTask[];
}
