export interface BuddyAvatarProps {
  className?: string;
}

export function BuddyAvatar({ className = 'h-8 w-8' }: BuddyAvatarProps) {
  return (
    <div
      className={`relative flex shrink-0 items-center justify-center rounded-xl bg-white p-1 shadow-xs ${className}`}
      aria-label="AI-buddy avatar"
    >
      <svg viewBox="0 0 35 35" fill="none" className="h-full w-full">
        <circle cx="12.5" cy="21.5" r="7.4" fill="#04E061" />
        <circle cx="25.4" cy="23.5" r="4.5" fill="#FF4053" />
        <circle cx="13.9" cy="10.4" r="2.8" fill="#965EEB" />
        <circle cx="23.7" cy="12.1" r="6.0" fill="#00AAFF" />
      </svg>
    </div>
  );
}

export default BuddyAvatar;
