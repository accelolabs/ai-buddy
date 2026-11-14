import { useNavigate } from 'react-router';

export function ChatHeader() {
  const navigate = useNavigate();

  return (
    <header className="flex w-full items-center justify-end px-6 py-4">
      <div className="flex h-10 w-60 items-center rounded-lg bg-[#E5E9F2] p-1">
        <button
          type="button"
          className="flex-1 rounded-md bg-white py-1.5 text-center text-sm font-semibold text-[#0F172A] shadow-sm transition-all"
        >
          Стажер
        </button>
        <button
          type="button"
          onClick={() => navigate('/dashboard')}
          className="flex-1 rounded-md py-1.5 text-center text-sm font-medium text-[#677891] transition-colors hover:text-[#0F172A]"
        >
          Наставник
        </button>
      </div>
    </header>
  );
}
