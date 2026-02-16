# KAN-75: Rest Timer - Architecture

## Scope

**Type**: UI-only feature (frontend)
**Backend changes**: None (`restSeconds` already in API response)
**Database changes**: None
**n8n changes**: None

## Data Flow

```
[Existing API Response]          [New Frontend Logic]
exercises[].restSeconds  --->  useRestTimer hook  --->  RestTimerOverlay
     (int, seconds)            (countdown state)        (fixed bottom bar)
```

## File Map

| Action | File | Lines Changed (est.) |
|--------|------|---------------------|
| CREATE | `src/hooks/useRestTimer.ts` | ~70 |
| CREATE | `src/components/RestTimerOverlay.tsx` | ~60 |
| MODIFY | `src/components/WorkoutContent.tsx` | ~20 (add hook call + trigger logic) |
| MODIFY | `src/index.css` | ~5 (slideUp keyframe) |

## Hook Interface

```typescript
// src/hooks/useRestTimer.ts
interface TimerState {
  isActive: boolean;
  remainingSeconds: number;
  totalSeconds: number;
  exerciseName: string;
  isFinished: boolean; // 3s "Tiempo!" state before auto-dismiss
}

function useRestTimer(): {
  timerState: TimerState;
  startTimer: (seconds: number, exerciseName: string) => void;
  dismissTimer: () => void;
}
```

### Hook Internals

- `useRef` for interval ID (clear on reset/unmount)
- `useEffect` manages 1s interval when `isActive === true`
- On reach 0: play beep via `AudioContext(800Hz, 0.3s)`, call `navigator.vibrate(300)` with feature check
- Set `isFinished = true`, then `setTimeout(3000)` to auto-dismiss
- `startTimer` called while active → clears old interval, restarts

### Audio (no dependencies)

```typescript
const ctx = new AudioContext();
const osc = ctx.createOscillator();
osc.frequency.value = 800;
osc.connect(ctx.destination);
osc.start();
osc.stop(ctx.currentTime + 0.3);
```

Wrapped in try/catch — fails silently if AudioContext unavailable.

## Trigger Logic (WorkoutContent.tsx)

In `handleToggleSet` and `handleToggleAltSet`:

```
IF set was incomplete AND is now complete
  AND exercise has restSeconds > 0
  AND NOT all sets of this exercise are now complete
THEN startTimer(restSeconds, exerciseName)
```

Key: read `wasCompleted` BEFORE `setExerciseList`, call `startTimer` AFTER.

## Component Props

```typescript
// src/components/RestTimerOverlay.tsx
interface RestTimerOverlayProps {
  remainingSeconds: number;
  totalSeconds: number;
  exerciseName: string;
  isFinished: boolean;
  onDismiss: () => void;
}
```

Rendered conditionally: `{timerState.isActive && <RestTimerOverlay ... />}`
