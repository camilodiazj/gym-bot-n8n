import { Timer, X } from "lucide-react";

interface RestTimerOverlayProps {
  remainingSeconds: number;
  totalSeconds: number;
  exerciseName: string;
  isFinished: boolean;
  onDismiss: () => void;
}

const formatTime = (seconds: number): string => {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}:${s.toString().padStart(2, "0")}`;
};

export const RestTimerOverlay = ({
  remainingSeconds,
  totalSeconds,
  exerciseName,
  isFinished,
  onDismiss,
}: RestTimerOverlayProps) => {
  const progressPercent = (remainingSeconds / totalSeconds) * 100;

  return (
    <div
      role="timer"
      aria-live="polite"
      aria-label={`Descanso: ${isFinished ? 'Tiempo' : formatTime(remainingSeconds)}`}
      className="fixed bottom-0 left-0 right-0 z-30 mx-auto max-w-[400px] rounded-t-2xl bg-[#1A1A1A] px-5 pt-4 pb-[max(1rem,env(safe-area-inset-bottom))] shadow-[0_-2px_16px_rgba(0,0,0,0.15)] animate-slideUp"
    >
      {/* Row 1: Timer + Countdown + Dismiss */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Timer className="h-5 w-5 text-[#22C55E]" />
          <span
            className={`font-['DM_Sans'] text-2xl font-bold leading-none tabular-nums ${
              isFinished ? "text-[#22C55E]" : "text-white"
            }`}
          >
            {isFinished ? "\u00a1Tiempo!" : formatTime(remainingSeconds)}
          </span>
        </div>
        <button
          onClick={onDismiss}
          onKeyDown={(e) => e.key === "Enter" && onDismiss()}
          aria-label="Saltar descanso"
          tabIndex={0}
          className="flex h-10 w-10 items-center justify-center rounded-lg text-gray-400 transition-colors hover:text-white hover:bg-white/10 active:scale-95"
        >
          <X className="h-5 w-5" />
        </button>
      </div>

      {/* Row 2: Progress bar */}
      <div className="mt-3 h-1 overflow-hidden rounded-full bg-gray-700">
        <div
          className="h-full rounded-full bg-[#22C55E] transition-all duration-[999ms] ease-linear"
          style={{ width: `${progressPercent}%` }}
        />
      </div>

      {/* Row 3: Exercise name */}
      <p className="mt-2 font-['DM_Sans'] text-xs text-gray-400 line-clamp-2">
        Descansando &middot; {exerciseName}
      </p>
    </div>
  );
};
