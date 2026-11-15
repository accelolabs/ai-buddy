import { useState, useRef, useEffect } from 'react';
import { useNavigate } from 'react-router';
import { useQueryClient } from '@tanstack/react-query';
import {
  User as UserIcon,
  ChevronDown,
  Hexagon,
  Building2,
  MessageSquare,
  ArrowRight,
  Check,
} from 'lucide-react';
import chipLogo from '@/assets/chip-logo.svg';
import { MOCK_USERS, type MockUserMeta, setUserIdCookie, getUserIdCookie } from '@/entities/user';

export function LoginForm() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const currentCookieId = getUserIdCookie();
  const initialUser = MOCK_USERS.find((u) => u.id === currentCookieId) ?? MOCK_USERS[0];

  const [selectedUser, setSelectedUser] = useState<MockUserMeta>(initialUser);
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleSelectUser = (user: MockUserMeta) => {
    setSelectedUser(user);
    setIsOpen(false);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setUserIdCookie(selectedUser.id);
    await queryClient.invalidateQueries({ queryKey: ['user'] });
    navigate(selectedUser.targetRoute);
  };

  return (
    <div className="relative w-full max-w-[440px]">
      <div className="mb-8 flex flex-col items-center text-center">
        <div className="mb-4 flex h-10 w-10 items-center justify-center">
          <img
            src={chipLogo}
            alt="AI-buddy logo"
            className="h-10 w-10 select-none drop-shadow-xs"
          />
        </div>

        <h1 className="text-[26px] font-extrabold tracking-tight text-[#0F172A] sm:text-[28px]">
          Вход в AI–buddy
        </h1>
        <p className="mt-1 text-[15px] font-normal text-[#64748B]">
          Выберите пользователя для входа
        </p>
      </div>

      <div className="relative rounded-2xl border border-slate-200/90 bg-white p-6 shadow-[0_8px_30px_rgb(15,23,42,0.04)] sm:p-7">
        <form onSubmit={handleSubmit} className="flex flex-col">
          <div className="mb-2 flex items-center gap-1.5 text-[13px] font-semibold text-[#475569]">
            <UserIcon className="h-3.5 w-3.5 stroke-[2]" />
            <span>Пользователь</span>
          </div>

          <div ref={dropdownRef} className="relative mb-5">
            <button
              type="button"
              onClick={() => setIsOpen((prev) => !prev)}
              aria-haspopup="listbox"
              aria-expanded={isOpen}
              className="flex h-12 w-full items-center justify-between rounded-xl border border-slate-200 bg-white px-3.5 py-2 text-left transition-all hover:border-slate-300 focus:border-[#00A3FF] focus:outline-hidden focus:ring-2 focus:ring-[#00A3FF]/20"
            >
              <div className="flex items-center gap-3">
                <div className="flex h-7 w-7 items-center justify-center rounded-full bg-[#00A3FF] text-[13px] font-bold text-white shadow-xs">
                  {selectedUser.name.charAt(0)}
                </div>
                <div className="flex items-center gap-2 text-[15px]">
                  <span className="font-bold text-[#0F172A]">{selectedUser.name}</span>
                  <span className="text-[#94A3B8]">—</span>
                  <span className="text-[14px] text-[#64748B]">{selectedUser.roleLabel}</span>
                </div>
              </div>

              <ChevronDown
                className={`h-4 w-4 text-[#94A3B8] transition-transform duration-200 ${
                  isOpen ? 'rotate-180' : ''
                }`}
              />
            </button>

            {isOpen && (
              <div
                role="listbox"
                className="absolute z-20 mt-1.5 w-full overflow-hidden rounded-xl border border-slate-200 bg-white py-1 shadow-lg"
              >
                {MOCK_USERS.map((user) => {
                  const isSelected = user.id === selectedUser.id;
                  return (
                    <button
                      key={user.id}
                      type="button"
                      role="option"
                      aria-selected={isSelected}
                      onClick={() => handleSelectUser(user)}
                      className={`flex w-full items-center justify-between px-3.5 py-2.5 text-left transition-colors hover:bg-slate-50 ${
                        isSelected ? 'bg-sky-50/60' : ''
                      }`}
                    >
                      <div className="flex items-center gap-3">
                        <div
                          className={`flex h-7 w-7 items-center justify-center rounded-full text-[13px] font-bold text-white shadow-xs ${
                            user.role === 'mentor' ? 'bg-purple-600' : 'bg-[#00A3FF]'
                          }`}
                        >
                          {user.name.charAt(0)}
                        </div>
                        <div className="flex items-center gap-2 text-[15px]">
                          <span className="font-bold text-[#0F172A]">{user.name}</span>
                          <span className="text-[#94A3B8]">—</span>
                          <span className="text-[14px] text-[#64748B]">{user.roleLabel}</span>
                        </div>
                      </div>

                      {isSelected && <Check className="h-4 w-4 text-[#00A3FF]" />}
                    </button>
                  );
                })}
              </div>
            )}
          </div>

          <div className="mb-5 grid grid-cols-2 gap-3">
            <div className="rounded-xl border border-slate-200 bg-white p-3">
              <div className="mb-1 flex items-center gap-1 text-[11.5px] font-medium text-[#64748B]">
                <Hexagon className="h-3 w-3 stroke-[2] text-[#94A3B8]" />
                <span>Роль</span>
              </div>
              <div className="text-[14px] font-bold text-[#0F172A]">{selectedUser.roleTitle}</div>
            </div>

            <div className="rounded-xl border border-slate-200 bg-white p-3">
              <div className="mb-1 flex items-center gap-1 text-[11.5px] font-medium text-[#64748B]">
                <Building2 className="h-3 w-3 stroke-[2] text-[#94A3B8]" />
                <span>Команда</span>
              </div>
              <div className="truncate text-[13px] font-bold text-[#0F172A]">
                {selectedUser.team}
              </div>
            </div>
          </div>

          <div className="mb-6 flex items-start gap-2.5 text-[11.5px] leading-relaxed text-[#64748B]">
            <MessageSquare className="mt-0.5 h-3.5 w-3.5 shrink-0 text-[#00A3FF]" />
            <p>{selectedUser.infoNote}</p>
          </div>

          <button
            type="submit"
            className="flex h-11 w-full items-center justify-center gap-1.5 rounded-xl bg-[#00A3FF] text-[15px] font-semibold text-white shadow-sm transition-all hover:bg-[#0092E6] active:scale-[0.99] focus:outline-hidden focus:ring-2 focus:ring-[#00A3FF]/40"
          >
            <span>Войти</span>
            <ArrowRight className="h-4 w-4 stroke-[2.5]" />
          </button>
        </form>
      </div>
    </div>
  );
}
