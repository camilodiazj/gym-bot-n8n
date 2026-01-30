// API Configuration
const API_BASE_URL = 'http://localhost:8080/api/v1';

/**
 * Updates the weight for a specific set
 * @param setId - Set ID in format "workoutId-setNumber"
 * @param kg - Weight value as string (e.g., "25", "30.5")
 */
export async function updateSetWeight(setId: string, kg: string): Promise<void> {
  const response = await fetch(`${API_BASE_URL}/sets/${setId}`, {
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
  const response = await fetch(`${API_BASE_URL}/sets/${setId}`, {
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
  const response = await fetch(`${API_BASE_URL}/sets/${setId}/complete`, {
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
