import { create } from 'zustand';
import type { TraineeIntern } from './types';
import { DEFAULT_INTERNS } from '../constants/defaultTasks';

interface RouteState {
  interns: TraineeIntern[];
  activeInternId: string;
  activeGlitchTaskId: string | null;

  // Actions
  setActiveIntern: (id: string) => void;
  toggleTask: (taskId: string) => void;
  retryTask: (taskId: string) => void;
  resetRoute: () => void;
}

export const useRouteStore = create<RouteState>((set, get) => ({
  interns: DEFAULT_INTERNS,
  activeInternId: 'anna',
  activeGlitchTaskId: null,

  setActiveIntern: (id: string) => {
    set({ activeInternId: id, activeGlitchTaskId: null });
  },

  toggleTask: (taskId: string) => {
    const { interns, activeInternId } = get();
    const activeIntern = interns.find((i) => i.id === activeInternId);
    if (!activeIntern) return;

    const task = activeIntern.tasks.find((t) => t.id === taskId);
    if (!task) return;

    // If it's the error task and currently has error, clicking retries or triggers glitch shake
    if (task.hasError && !task.completed) {
      set({
        activeGlitchTaskId: taskId,
        interns: interns.map((intern) =>
          intern.id === activeInternId
            ? {
                ...intern,
                tasks: intern.tasks.map((t) =>
                  t.id === taskId
                    ? {
                        ...t,
                        isGlitching: true,
                      }
                    : t,
                ),
              }
            : intern,
        ),
      });

      // Clear glitch shake after 1.5s
      setTimeout(() => {
        set((state) => ({
          activeGlitchTaskId: null,
          interns: state.interns.map((intern) =>
            intern.id === state.activeInternId
              ? {
                  ...intern,
                  tasks: intern.tasks.map((t) =>
                    t.id === taskId ? { ...t, isGlitching: false } : t,
                  ),
                }
              : intern,
          ),
        }));
      }, 1500);
      return;
    }

    const nextCompleted = !task.completed;
    const nextStatus = nextCompleted ? 'completed' : 'pending';
    const nextSubtitle = nextCompleted ? 'Завершено' : 'Ожидает';

    set({
      interns: interns.map((intern) =>
        intern.id === activeInternId
          ? {
              ...intern,
              tasks: intern.tasks.map((t) =>
                t.id === taskId
                  ? {
                      ...t,
                      completed: nextCompleted,
                      status: nextStatus,
                      subtitle: nextSubtitle,
                      hasError: false,
                      errorMessage: undefined,
                      isGlitching: false,
                    }
                  : t,
              ),
            }
          : intern,
      ),
    });
  },

  retryTask: (taskId: string) => {
    const { interns, activeInternId } = get();

    set({
      activeGlitchTaskId: null,
      interns: interns.map((intern) =>
        intern.id === activeInternId
          ? {
              ...intern,
              tasks: intern.tasks.map((t) =>
                t.id === taskId
                  ? {
                      ...t,
                      completed: true,
                      status: 'completed',
                      subtitle: 'Завершено',
                      hasError: false,
                      errorMessage: undefined,
                      isGlitching: false,
                    }
                  : t,
              ),
            }
          : intern,
      ),
    });
  },

  resetRoute: () => {
    set({
      interns: DEFAULT_INTERNS,
      activeInternId: 'anna',
      activeGlitchTaskId: null,
    });
  },
}));

export const selectActiveIntern = (state: RouteState): TraineeIntern => {
  return (
    state.interns.find((i) => i.id === state.activeInternId) ||
    state.interns[0]
  );
};

export const selectCompletedCount = (state: RouteState): number => {
  const intern = selectActiveIntern(state);
  return intern.tasks.filter((t) => t.completed).length;
};

export const selectTotalCount = (state: RouteState): number => {
  const intern = selectActiveIntern(state);
  return intern.tasks.length;
};

export const selectProgressPercentage = (state: RouteState): number => {
  const completed = selectCompletedCount(state);
  const total = selectTotalCount(state);
  if (total === 0) return 0;
  return Math.round((completed / total) * 100);
};

export const selectFilledSegments = (state: RouteState): number => {
  const completed = selectCompletedCount(state);
  const total = selectTotalCount(state);
  if (total === 0) return 0;
  return Math.round((completed / total) * 24);
};
