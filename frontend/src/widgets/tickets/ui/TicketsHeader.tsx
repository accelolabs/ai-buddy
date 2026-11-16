import { ArrowLeft, LogOut } from 'lucide-react';
import { useNavigate } from 'react-router';
import { clearUserIdCookie } from '@/entities/user';

export interface TicketsHeaderProps {
  onBack?: () => void;
  onLogout?: () => void;
}

export function TicketsHeader({ onBack, onLogout }: TicketsHeaderProps) {
  const navigate = useNavigate();

  const handleBack = () => {
    if (onBack) {
      onBack();
    } else {
      navigate('/');
    }
  };

  const handleLogout = () => {
    if (onLogout) {
      onLogout();
    } else {
      clearUserIdCookie();
      navigate('/login');
    }
  };

  return (
    <header className="mb-6 flex items-center justify-between gap-3">
      <div className="flex items-center gap-3">
        <button
          type="button"
          onClick={handleBack}
          className="inline-flex h-9 items-center gap-2 rounded-[8px] border border-slate-200 bg-white px-3.5 text-xs font-semibold text-slate-800 shadow-xs transition-colors hover:bg-slate-50 hover:text-slate-900 focus-visible:outline focus-visible:outline-2 focus-visible:outline-sky-500"
        >
          <ArrowLeft className="h-4 w-4 shrink-0 text-slate-800" />
          <span>Вернуться в чат</span>
        </button>

        <button
          type="button"
          onClick={handleLogout}
          className="inline-flex h-9 items-center gap-2 rounded-[8px] border border-slate-200 bg-white px-3.5 text-xs font-semibold text-slate-600 shadow-xs transition-colors hover:bg-slate-50 hover:text-slate-900 focus-visible:outline focus-visible:outline-2 focus-visible:outline-sky-500"
        >
          <LogOut className="h-4 w-4 shrink-0 text-slate-500" />
          <span>Выйти</span>
        </button>
      </div>
    </header>
  );
}

export default TicketsHeader;
