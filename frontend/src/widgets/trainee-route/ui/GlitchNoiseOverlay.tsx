import { memo } from 'react';
import { AlertTriangle, RotateCw } from 'lucide-react';

export interface GlitchNoiseOverlayProps {
  message?: string;
  onRetry?: () => void;
  isGlitching?: boolean;
}

// Authentic static noise pattern from mush.svg
const MUSH_BASE64 =
  'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAt8AAADaCAYAAAB+WC5eAAB+3klEQVR4nOydB3gVRdfH/7e3tJtKQholhN5770gRRRFQFMWCICiIvX+vBQUrICiCFCsdpPcWei8JJARSSU9u+u3le2b23pDQBOXGoOf35D6b3dmdcubs7JnZM7NSo9nqwE0wW2ywOxxQyqVwNyXlJnhpFG5Px2CyQCoWQyaTuDUdq80Gk9kGjUoOd1NmMEOtkEIsFrs1HbPZBjvcrw9MIUvLjfDSKOFu9EYzZFIpZFL3ys5qs8NssUKtdL8+lOpNXO/EIpFb0zGarWBJKGTu1Qe73cHryUPt/vah3GCGQi6FVOJefbBY7byNUClkdyW+rgk/Vvy/P/qZKmElZUZ4ahQQuVkfDGYLJCIx5G5uW+02Owxma7W0rXqjBQq5BBI3t60Wi03Qh2poH5g+eHm4v201miz8meR2fbDbYTBVjz6w9kGllEIscvOzltledjuUd6l9uBUl1fSsNRgtkEklkLr9WWurNtvr7zxr3SsFgiAIgiAIgiAqIOObIAiCIAiCIKoJMr4JgiAIgiAIopog45sgCIIgCIIgqgkyvgmCIAiCIAiimiDjmyAIgiAIgiCqCTK+CYIgCIIgCKKaIOObIAiCIAiCIKoJMr4JgiAIgiAIopog45sgCIIgCIIgqgkyvgmCIAiCIAiimiDjmyAIgiAIgiCqCTK+CYIgCIIgCKKaIOObIAiCIAiCIKoJaXUlRBAEQdx9JGIrjFqR8G+CIIh/O5o9/1sAedP9H0EQBEEQBEEQ1eKfZHzT5g9BEARBEEQR44H5V5v8k92bIAiCIAiCKA7I8CYIgiAIgiCIajC+yccb/08x/u33BcTYJwiCIAiCIIhfB4q3TRuO30Tct53+hF9D1CR+DVEf/8+3A/9j7cD/F+ObjG+CIAiCIAiC+H+A8U2G9/8bxH/yfcG/6H6g7wHUHlB7+I9vB6gd+KeNbzK8CYIgCIIgCOJfgxXU/v8Q/1z/8gQhCIIgCIIgCEfAYbG7Z47f/0e2nSAIgiAIgiDuLRSXmyAIgiAIgiDsBBnfBEEQBEEQBGEnyPgmCIIgCIIgCDtBxjdBEARBEARB2AkyvgmCIAiCIAjCTpDxTRAEQRAEQRB2goxvgiAIgiAIgrATZHwTBEEQBEEQhJ0g45sgCIIgCIIg7AQZ3wRBEARBEARhJ8j4JgiCIAiCIAg7QcY3QRAEQRAEQdgJMr4JgiAIgiAIwk6Q8U0QBEEQBEEQdoKMb4IgCIIgCIKwE2R8EwRBEARBEISdIOObIAiCIAiCIKoR/x8gG0hP5VjD3QAAAABJRU5ErkJggg==';

export const GlitchNoiseOverlay = memo(function GlitchNoiseOverlay({
  message = 'Не удалось подтвердить выполнение задания в Tracker. Помехи на линии связи.',
  onRetry,
  isGlitching = false,
}: GlitchNoiseOverlayProps) {
  return (
    <div className="relative mt-3 overflow-hidden rounded-xl border border-rose-200 bg-rose-50/90 p-3.5 text-rose-950 shadow-inner">
      {/* Static Noise Texture Overlay from mush.svg */}
      <div
        className={`pointer-events-none absolute inset-0 z-10 mix-blend-multiply transition-opacity duration-200 ${
          isGlitching ? 'animate-noise-flicker opacity-90' : 'opacity-40'
        }`}
        style={{
          backgroundImage: `url(${MUSH_BASE64})`,
          backgroundRepeat: 'repeat',
          backgroundSize: '360px 110px',
        }}
      />

      {/* CRT Scanline Overlay */}
      <div className="pointer-events-none absolute inset-0 z-10 scanlines-overlay opacity-30" />

      {/* Jitter Glitch Indicator */}
      {isGlitching && (
        <div className="pointer-events-none absolute inset-0 z-20 border-2 border-rose-500/80 animate-glitch-shake" />
      )}

      {/* Content */}
      <div className="relative z-20 flex flex-col gap-2.5 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-start gap-2.5">
          <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-rose-500/15 text-rose-600">
            <AlertTriangle className="h-4 w-4" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="text-xs font-semibold tracking-wide uppercase text-rose-600">
                Сбой связи • Помехи
              </span>
              {isGlitching && (
                <span className="inline-flex items-center rounded-full bg-rose-500 px-1.5 py-0.2 text-[10px] font-bold text-white uppercase animate-pulse">
                  GLITCH
                </span>
              )}
            </div>
            <p className="mt-0.5 text-xs text-rose-900/90 leading-relaxed font-mono">
              {message}
            </p>
          </div>
        </div>

        {onRetry && (
          <button
            type="button"
            onClick={onRetry}
            className="inline-flex shrink-0 items-center justify-center gap-1.5 self-start sm:self-center rounded-lg border border-rose-300 bg-white/90 px-3 py-1.5 text-xs font-semibold text-rose-800 shadow-sm transition-all duration-150 hover:bg-rose-100 hover:text-rose-900 active:scale-95 cursor-pointer"
          >
            <RotateCw className="h-3.5 w-3.5" />
            <span>Повторить</span>
          </button>
        )}
      </div>
    </div>
  );
});

export default GlitchNoiseOverlay;
