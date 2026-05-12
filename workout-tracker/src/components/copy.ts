/**
 * UI copy catalog for the workout tracker. Spanish (es-CO).
 *
 * No i18n framework needed today — just a plain typed object that keeps
 * literal strings out of JSX. When the day comes to support more locales,
 * this file becomes the `es-CO` catalog of a react-i18next setup without
 * touching component code.
 *
 * Grouped by component so unrelated copy doesn't accidentally couple.
 */
export const copy = {
  workoutContent: {
    /** Badge above the exercise name in the workout card. */
    exerciseBadge: "Ejercicio",
    /** Header of the collapsible instructions section. */
    instructionsHeading: "Instrucciones",
  },
} as const;

export type Copy = typeof copy;
