import { useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useUser, type User } from '@/entities/user';
import { request, queryClient } from '@/shared/api';
import { sessionSignal } from '@/shared/api/session';
import type { Task } from '@/shared/api/contracts';
import { RequestState } from '@/shared/ui/RequestState';
import { RouteProgressBar, RouteChamferCard, RouteWireSegment } from '@/widgets/trainee-route';

export function RoutePage() {
  const { data: user } = useUser();
  const mentor = user?.role === 'mentor';
  const [selected, setSelected] = useState('');
  const interns = useQuery({
    queryKey: ['interns', user?.id],
    queryFn: ({ signal }) => request<User[]>('/api/users/interns', { signal }),
    enabled: mentor,
  });
  const intern = mentor
    ? (interns.data?.find((u) => u.id === selected) ?? interns.data?.[0])
    : user;
  const tasks = useQuery({
    queryKey: ['tasks', user?.id, intern?.id],
    queryFn: ({ signal }) =>
      request<Task[]>(
        mentor
          ? `/api/tracker/interns/${encodeURIComponent(intern!.id)}/tasks`
          : '/api/tracker/tasks',
        { signal },
      ),
    enabled: !!intern,
  });
  const complete = useMutation({
    mutationFn: (id: string) =>
      request<Task>(`/api/tracker/tasks/${encodeURIComponent(id)}/complete`, {
        method: 'POST',
        signal: sessionSignal(),
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['tasks'] }),
  });
  return (
    <div className="min-h-full flex-1 bg-[#F8FAFC] p-4 sm:p-6 lg:p-8">
      <div className="mx-auto max-w-[942px] pb-12">
        <h1 className="mb-6 text-[32px] font-bold text-slate-900">
          {mentor ? 'Мои стажеры' : 'Мой маршрут'}
        </h1>
        {mentor && (
          <>
            <RequestState
              loading={interns.isPending}
              error={interns.error}
              retry={() => void interns.refetch()}
              empty={
                !interns.isPending && !interns.data?.length ? 'Нет назначенных стажёров' : undefined
              }
            />
            <div className="mb-6 flex flex-wrap gap-3">
              {interns.data?.map((u) => (
                <button
                  key={u.id}
                  className={`rounded-xl border px-4 py-2 ${u.id === intern?.id ? 'border-sky-300 bg-sky-50 text-sky-900' : 'bg-white'}`}
                  onClick={() => {
                    setSelected(u.id);
                    complete.reset();
                  }}
                >
                  {u.name}
                </button>
              ))}
            </div>
          </>
        )}
        {intern && (
          <div className="rounded-3xl border border-slate-200 bg-white p-6 shadow-xs sm:p-8">
            <p className="text-xs font-bold uppercase text-sky-600">Стажёр · {intern.team}</p>
            <h2 className="mb-6 mt-1 text-2xl font-bold">{intern.name}</h2>
            <RequestState
              loading={tasks.isPending}
              error={tasks.error}
              retry={() => void tasks.refetch()}
              empty={tasks.isSuccess && !tasks.data.length ? 'Задач пока нет' : undefined}
            />
            {tasks.data && (
              <>
                <RouteProgressBar
                  completedCount={tasks.data.filter((t) => t.status === 'completed').length}
                  totalCount={tasks.data.length}
                />
                {tasks.data.map((task, i) => (
                  <div key={task.id}>
                    <RouteWireSegment variant={task.status === 'completed' ? 'green' : 'gray'} />
                    <RouteChamferCard
                      task={task}
                      order={i + 1}
                      pending={complete.isPending}
                      onComplete={
                        !mentor
                          ? () => {
                              if (!complete.isPending) complete.mutate(task.id);
                            }
                          : undefined
                      }
                    />
                  </div>
                ))}
              </>
            )}
            {complete.error && (
              <p role="alert" className="mt-4 text-red-700">
                {complete.error.message}
              </p>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
