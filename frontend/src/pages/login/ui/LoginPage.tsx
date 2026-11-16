import { LoginForm } from './LoginForm';

export function LoginPage() {
  return (
    <div className="relative flex min-h-screen w-full items-center justify-center overflow-hidden bg-white px-4 py-8 sm:py-12">
      <div className="pointer-events-none absolute inset-0 flex items-center justify-center overflow-hidden">
        <svg
          className="h-[588px] w-[500px] shrink-0 text-slate-200"
          viewBox="0 0 500 588"
          fill="none"
          xmlns="http://www.w3.org/2000/svg"
        >
          <path d="M 500 264 H 336 V 334" stroke="currentColor" strokeWidth="1.2" />
          <circle cx="336" cy="334" r="3" fill="#CBD5E1" />

          <path d="M 456 498 H 500" stroke="currentColor" strokeWidth="1.2" />

          <path d="M 359 550 V 588" stroke="currentColor" strokeWidth="1.2" />
        </svg>
      </div>

      <LoginForm />
    </div>
  );
}

export default LoginPage;
