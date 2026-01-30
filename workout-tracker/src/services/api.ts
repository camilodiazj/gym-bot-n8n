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
