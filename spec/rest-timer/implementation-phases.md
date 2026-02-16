# KAN-75: Rest Timer - Implementation Phases

## Phase 1: Core Timer Hook (pixel-dev)

**File**: `workout-tracker/src/hooks/useRestTimer.ts` (~70 lines)

Create the `useRestTimer` custom hook:
- State: `isActive`, `remainingSeconds`, `totalSeconds`, `exerciseName`, `isFinished`
- `startTimer(seconds, name)`: clears existing interval, sets state, starts new 1s interval
- `dismissTimer()`: clears interval, resets all state
- On reach 0: play beep (`AudioContext`), vibrate (`navigator.vibrate(300)`), set `isFinished=true`, auto-dismiss after 3s
- Cleanup on unmount

**Depends on**: Nothing (can start immediately)

## Phase 2: Timer Overlay Component (pixel-dev)

**File**: `workout-tracker/src/components/RestTimerOverlay.tsx` (~60 lines)

Build the fixed bottom bar per [ux-ui-spec.md](./ux-ui-spec.md):
- Props: `remainingSeconds`, `totalSeconds`, `exerciseName`, `isFinished`, `onDismiss`
- Format time as `M:SS`
- Progress bar width = `(remaining/total) * 100%`
- Conditional render: countdown or "Tiempo!"

**File**: `workout-tracker/src/index.css` (~5 lines added)

Add `slideUp` keyframe animation.

**Depends on**: Nothing (can run parallel with Phase 1)

## Phase 3: Integration (pixel-dev)

**File**: `workout-tracker/src/components/WorkoutContent.tsx` (~20 lines changed)

Wire hook + overlay into existing component:

1. Import `useRestTimer` and `RestTimerOverlay`
2. Call `useRestTimer()` at component level
3. In `handleToggleSet` (~line 676):
   - Before `setExerciseList`: capture `wasCompleted` from current state
   - After `setExerciseList`: if `!wasCompleted && restSeconds > 0 && !allSetsNowComplete` → `startTimer(restSeconds, name)`
4. Same logic in `handleToggleAltSet` (~line 758) using alt exercise data
5. Render `<RestTimerOverlay>` at bottom of JSX, conditional on `timerState.isActive`

**Depends on**: Phase 1 + Phase 2

## Phase 4: QA (manual)

Verify using `npm run dev` + `?demo` URL param (demo data already has `restSeconds: 120` and `90`):

| Test | Expected |
|------|----------|
| Complete set 1 of 3 | Timer starts, shows exercise name, countdown from restSeconds |
| Timer reaches 0 | Beep + vibration + "Tiempo!" for 3s + auto-dismiss |
| Tap X during countdown | Timer dismisses immediately |
| Complete set 2 while timer running | Timer resets with new time |
| Complete set 3 (last) | NO timer (exercise collapses, scrolls to next) |
| Complete alt exercise set | Timer uses alt exercise's restSeconds and name |
| `npm test` | All existing tests pass |

## Parallelization

```
Phase 1 (hook)  ──┐
                   ├──> Phase 3 (integration) ──> Phase 4 (QA)
Phase 2 (UI)   ──┘
```

Phases 1 and 2 are independent and can be built simultaneously.

## Roles

| Phase | Agent | Why |
|-------|-------|-----|
| 1-3 | **pixel-dev** | Pure frontend React/TypeScript work |
| 4 | **code-reviewer** | Post-implementation audit |

No involvement needed from: n8n-agent, kiro-coach, claude-designer (spec already done).
