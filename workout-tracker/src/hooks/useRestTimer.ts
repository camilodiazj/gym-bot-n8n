import { useState, useRef, useCallback, useEffect } from 'react';

interface TimerState {
  isActive: boolean;
  remainingSeconds: number;
  totalSeconds: number;
  exerciseName: string;
  isFinished: boolean;
}

const INITIAL_STATE: TimerState = {
  isActive: false,
  remainingSeconds: 0,
  totalSeconds: 0,
  exerciseName: '',
  isFinished: false,
};

function playBeep(): void {
  try {
    const ctx = new AudioContext();
    const osc = ctx.createOscillator();
    osc.frequency.value = 800;
    osc.connect(ctx.destination);
    osc.start();
    osc.stop(ctx.currentTime + 0.3);
  } catch {
    // fail silently
  }
}

export function useRestTimer() {
  const [timerState, setTimerState] = useState<TimerState>(INITIAL_STATE);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const clearTimers = useCallback(() => {
    if (intervalRef.current) clearInterval(intervalRef.current);
    if (timeoutRef.current) clearTimeout(timeoutRef.current);
    intervalRef.current = null;
    timeoutRef.current = null;
  }, []);

  const dismissTimer = useCallback(() => {
    clearTimers();
    setTimerState(INITIAL_STATE);
  }, [clearTimers]);

  const startTimer = useCallback((seconds: number, exerciseName: string) => {
    clearTimers();
    setTimerState({
      isActive: true,
      remainingSeconds: seconds,
      totalSeconds: seconds,
      exerciseName,
      isFinished: false,
    });
  }, [clearTimers]);

  useEffect(() => {
    if (!timerState.isActive || timerState.isFinished) return;

    intervalRef.current = setInterval(() => {
      setTimerState((prev) => {
        if (prev.remainingSeconds <= 1) {
          clearInterval(intervalRef.current!);
          intervalRef.current = null;
          playBeep();
          try { if ('vibrate' in navigator) navigator.vibrate(300); } catch { /* iOS Safari */ }
          timeoutRef.current = setTimeout(dismissTimer, 3000);
          return { ...prev, remainingSeconds: 0, isFinished: true };
        }
        return { ...prev, remainingSeconds: prev.remainingSeconds - 1 };
      });
    }, 1000);

    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, [timerState.isActive, timerState.isFinished, dismissTimer]);

  useEffect(() => {
    return clearTimers;
  }, [clearTimers]);

  return { timerState, startTimer, dismissTimer } as const;
}
