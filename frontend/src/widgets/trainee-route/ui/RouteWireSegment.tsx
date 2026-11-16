export function RouteWireSegment({ variant }: { variant: 'green' | 'gray' }) {
  return (
    <div className="pointer-events-none flex h-8 justify-center">
      <svg width="2" height="32" viewBox="0 0 2 32">
        <rect width="2" height="32" fill={variant === 'green' ? '#34D399' : '#CBD5E1'} />
      </svg>
    </div>
  );
}
