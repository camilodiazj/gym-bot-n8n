# KAN-75: Rest Timer - UX/UI Spec

## Component: RestTimerOverlay

Fixed bottom bar (like a music player mini-bar). Non-blocking — user can scroll and interact with exercises above it.

## Layout

```
┌─────────────────────────────────────────┐
│  [Timer]  1:30          [X]             │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  │  ← progress bar
│  Descansando · Press de Banca           │
└─────────────────────────────────────────┘
  ▲ ~80px + safe-area-inset-bottom
```

## Specs

| Element | Value |
|---------|-------|
| Position | `fixed bottom-0 left-0 right-0`, centered `max-w-[400px] mx-auto` |
| Background | `#1A1A1A` |
| Corner radius | `rounded-t-2xl` |
| Padding | `px-5 pt-4 pb-4` + `pb-[env(safe-area-inset-bottom)]` |
| Z-index | `30` |
| Shadow | `shadow-[0_-2px_16px_rgba(0,0,0,0.15)]` |

### Row 1: Timer + Dismiss

| Element | Spec |
|---------|------|
| Timer icon | Lucide `Timer`, 20px, `#22C55E` |
| Countdown text | `font-['DM_Sans'] font-bold leading-none text-2xl text-white tabular-nums` |
| Format | `M:SS` (e.g., `1:30`, `0:45`) |
| Dismiss button | Lucide `X`, 20px, `text-gray-400`, tap target `40x40px`, `tabIndex={0}` |

### Row 2: Progress Bar

| Element | Spec |
|---------|------|
| Track | `h-1 bg-gray-700 rounded-full` |
| Fill | `bg-[#22C55E] rounded-full`, width = `(remaining/total) * 100%` |
| Transition | `transition-all duration-[999ms] ease-linear` |

### Row 3: Exercise Name

| Element | Spec |
|---------|------|
| Text | `font-['DM_Sans'] text-xs text-gray-400` |
| Format | `Descansando · {exerciseName}` |
| Overflow | `line-clamp-2` (allow 2 lines for long names) |

## Content Bottom Padding

When timer is active, the workout content container must add extra bottom padding to prevent the "Completar Rutina" button from being hidden behind the timer overlay:

```
pb-[calc(88px+env(safe-area-inset-bottom))]  // when timer active
pb-8                                          // when timer inactive (current)
```

## States

### Active Countdown
- Bar slides up (`translateY(100%) → translateY(0)`, 300ms `cubic-bezier(0.4, 0, 0.2, 1)`)
- Countdown decrements every second
- Progress bar shrinks left-to-right

### Finished (3 seconds)
- Countdown text replaced with `¡Tiempo!` in green (`#22C55E`)
- Beep sound (AudioContext 800Hz, 0.3s)
- Vibration: `if ('vibrate' in navigator) navigator.vibrate(300)` (no-op on iOS)
- Auto-dismiss after 3s

### Dismiss
- Slide down animation (`translateY(0) → translateY(100%)`, 300ms `cubic-bezier(0.4, 0, 0.2, 1)`)
- Triggered by X button OR auto-dismiss after "Tiempo!"

### Timer Reset (new set completed while active)
- No re-entrance animation — update in-place
- Exercise name fades to new name (200ms transition)
- Countdown resets to new `restSeconds` value

## Accessibility

- Container: `role="timer"`, `aria-live="polite"`
- Dismiss button: `aria-label="Saltar descanso"`, `tabIndex={0}`, `onKeyDown` handles Enter
- Contrast: white on `#1A1A1A` = 14.5:1 (WCAG AAA)

## Edge Cases

| Scenario | Behavior |
|----------|----------|
| `restSeconds` is 0 or undefined | No timer starts |
| Last set of current exercise completed (but more exercises remain) | No timer — auto-scroll/collapse takes over |
| Last set of last exercise completed | No timer |
| Non-last set completed on any exercise | Timer starts |
| New set completed while timer active | Timer resets in-place with new exercise's time |
| User un-completes a set | Timer unaffected |
| Phone locks/app backgrounds | Interval pauses naturally, resumes on return |
| `restSeconds < 10` | Timer starts normally (no minimum enforced) |

## Design Review Status

- **Reviewed by**: claude-designer
- **Status**: APPROVED (after fixes applied)
- **Fixes applied**: background color (#1A1A1A), typography (font-bold leading-none), z-index (30), bottom padding conflict, last-exercise logic clarification, keyboard accessibility
