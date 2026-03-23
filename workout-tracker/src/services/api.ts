import config from '../config';

const API_BASE_URL = config.apiBaseUrl;

/**
 * Gets the auth query string from URL (?c=code) or falls back to dev user_id
 */
function getAuthParams(): string {
  const params = new URLSearchParams(window.location.search);
  const code = params.get('c');

  if (code) {
    return `?c=${encodeURIComponent(code)}`;
  } else if (config.devUserId) {
    return `?user_id=${config.devUserId}`;
  }
  return '';
}

/**
 * Updates the weight for a specific set
 * @param setId - Set ID in format "workoutId-setNumber"
 * @param kg - Weight value as string (e.g., "25", "30.5")
 */
export async function updateSetWeight(setId: string, kg: string): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/sets/${setId}${getAuthParams()}`, {
    method: 'PATCH',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ kg }),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.message || `Failed to update set: ${response.statusText}`);
  }
}

/**
 * Updates the reps for a specific set
 * @param setId - Set ID in format "workoutId-setNumber"
 * @param reps - Number of reps performed
 */
export async function updateSetReps(setId: string, reps: number): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/sets/${setId}${getAuthParams()}`, {
    method: 'PATCH',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ reps }),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.message || `Failed to update set: ${response.statusText}`);
  }
}

/**
 * Marks a set as completed
 * @param setId - Set ID in format "workoutId-setNumber"
 */
export async function markSetComplete(setId: string): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/sets/${setId}/complete${getAuthParams()}`, {
    method: 'PATCH',
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.message || `Failed to mark set complete: ${response.statusText}`);
  }
}

/**
 * Marks a workout session as completed
 * @param sessionId - The day_routine_id from user_weekly_schedule
 */
export async function completeWorkout(sessionId: string): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/workouts/${sessionId}/complete${getAuthParams()}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.message || `Failed to complete workout: ${response.statusText}`);
  }
}

// ---------------------------------------------------------------------------
// Draft Routine API
// ---------------------------------------------------------------------------

export interface DraftAlternative {
  exerciseId: string;
  name: string;
  muscle: string;
  videoLink?: string;
}

export interface DraftExercise {
  exerciseId: string;
  exerciseOrder: number;
  name: string;
  muscle: string;
  sets: number;
  reps: string;
  rir: string;
  restSeconds: number;
  videoLink?: string;
  alternatives: DraftAlternative[];
}

export interface DraftDay {
  dayNumber: number;
  title: string;
  exercises: DraftExercise[];
}

export interface DraftData {
  code: string;
  status: 'pending' | 'approved' | 'expired';
  goal: string;
  level: string;
  daysPerWeek: number;
  scheduleLabel: string;
  days: DraftDay[];
  expiresAt: string;
}

export interface DraftResponse {
  success: boolean;
  data?: DraftData;
  error?: { code: number; message: string };
}

/**
 * Fetches a draft routine by its short code.
 * @param code - The draft access code from URL param ?c=
 */
// Map backend snake_case response to frontend camelCase interfaces
function mapDraftResponse(raw: Record<string, unknown>): DraftResponse {
  if (!raw.success || !raw.data) {
    return { success: false, error: raw.error as DraftResponse['error'] };
  }

  const d = raw.data as Record<string, unknown>;
  const days = (d.days as Record<string, unknown>[]) ?? [];

  const SCHEDULE_LABELS: Record<string, string> = {
    fb_2: 'Full Body 2 dias',
    fb_3: 'Cuerpo Completo (Full Body)',
    ul_4: 'Upper/Lower 4 dias',
    ppl_5: 'Push/Pull/Legs 5 dias',
    ppl_6: 'Push/Pull/Legs 6 dias',
  };

  const ws = (d.week_schedule as string) ?? 'fb_3';

  return {
    success: true,
    data: {
      code: (d.code as string) ?? '',
      status: (d.status as DraftData['status']) ?? 'pending',
      goal: (d.goal as string) ?? '',
      level: (d.level as string) ?? '',
      daysPerWeek: days.length,
      scheduleLabel: SCHEDULE_LABELS[ws] ?? ws,
      expiresAt: (d.expires_at as string) ?? '',
      days: days.map((day: Record<string, unknown>) => ({
        dayNumber: (day.day_number as number) ?? 0,
        title: (day.title as string) ?? '',
        exercises: ((day.exercises as Record<string, unknown>[]) ?? []).map(
          (ex: Record<string, unknown>) => ({
            exerciseId: (ex.exercise_id as string) ?? '',
            exerciseOrder: (ex.exercise_order as number) ?? 0,
            name: (ex.spanish_name as string) ?? '',
            muscle: (ex.main_muscle as string) ?? '',
            sets: (ex.sets as number) ?? 3,
            reps: (ex.reps as string) ?? '',
            rir: (ex.rir as string) ?? '',
            restSeconds: (ex.rest_seconds as number) ?? 90,
            videoLink: (ex.video_link as string) ?? (ex.link as string) ?? '',
            alternatives: ((ex.alternatives as Record<string, unknown>[]) ?? []).map(
              (alt: Record<string, unknown>) => ({
                exerciseId: (alt.exercise_id as string) ?? '',
                name: (alt.spanish_name as string) ?? '',
                muscle: (alt.main_muscle as string) ?? '',
                videoLink: (alt.video_link as string) ?? (alt.link as string) ?? '',
              })
            ),
          })
        ),
      })),
    },
  };
}

export async function getDraft(code: string): Promise<DraftResponse> {
  const response = await fetch(`${API_BASE_URL}/drafts/${encodeURIComponent(code)}`);

  if (response.status === 404 || response.status === 410) {
    const errorData = await response.json().catch(() => ({}));
    return {
      success: false,
      error: { code: response.status, message: errorData.message || 'Borrador no encontrado o expirado' },
    };
  }

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.message || `Failed to fetch draft: ${response.statusText}`);
  }

  const raw = await response.json();
  return mapDraftResponse(raw);
}

/**
 * Swaps an exercise in the draft with one of its alternatives.
 * @param code - The draft access code
 * @param dayNumber - Day number (1-indexed)
 * @param exerciseOrder - The exercise_order position within the day
 * @param newExerciseId - The exercise_id of the chosen alternative
 */
export async function swapDraftExercise(
  code: string,
  dayNumber: number,
  exerciseOrder: number,
  newExerciseId: string,
): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/drafts/${encodeURIComponent(code)}/swap`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ day_number: dayNumber, exercise_order: exerciseOrder, new_exercise_id: newExerciseId }),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.message || `Failed to swap exercise: ${response.statusText}`);
  }
}

/**
 * Approves a draft routine, triggering workout creation on the backend.
 * @param code - The draft access code
 */
export async function approveDraft(code: string): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/drafts/${encodeURIComponent(code)}/approve`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.message || `Failed to approve draft: ${response.statusText}`);
  }
}
