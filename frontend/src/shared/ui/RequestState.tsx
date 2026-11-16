export function RequestState({
  loading,
  error,
  retry,
  empty,
}: {
  loading?: boolean;
  error?: Error | null;
  retry?: () => void;
  empty?: string;
}) {
  if (loading)
    return (
      <p role="status" className="p-6 text-slate-500">
        Загрузка…
      </p>
    );
  if (error)
    return (
      <div role="alert" className="rounded-xl border border-red-200 bg-red-50 p-4 text-red-800">
        <p>{error.message}</p>
        <button className="mt-2 underline" onClick={retry}>
          Повторить
        </button>
      </div>
    );
  return empty ? <p className="p-6 text-slate-500">{empty}</p> : null;
}
