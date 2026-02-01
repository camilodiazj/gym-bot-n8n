/**
 * Timezone utilities for GymBot frontend.
 * All dates from the API are in UTC. These utilities convert for display.
 */

/** User's timezone (Colombia) */
export const USER_TIMEZONE = 'America/Bogota';

/** Locale for formatting dates in Spanish (Colombia) */
export const USER_LOCALE = 'es-CO';

/**
 * Formats a UTC date string for display in the user's timezone.
 * @param utcDate - ISO 8601 date string in UTC (e.g., "2026-01-31T05:00:00Z")
 * @returns Formatted date string in Spanish (e.g., "viernes, 31 de enero de 2026")
 */
export function formatDateForUser(utcDate: string): string {
  const date = new Date(utcDate);
  return date.toLocaleDateString(USER_LOCALE, {
    timeZone: USER_TIMEZONE,
    weekday: 'long',
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });
}

/**
 * Formats a UTC date string to show only the short date.
 * @param utcDate - ISO 8601 date string in UTC
 * @returns Short date string (e.g., "31/1/2026")
 */
export function formatShortDateForUser(utcDate: string): string {
  const date = new Date(utcDate);
  return date.toLocaleDateString(USER_LOCALE, {
    timeZone: USER_TIMEZONE,
    year: 'numeric',
    month: 'numeric',
    day: 'numeric',
  });
}

/**
 * Formats a UTC date string to show only the time.
 * @param utcDate - ISO 8601 date string in UTC
 * @returns Time string (e.g., "7:30 p.m.")
 */
export function formatTimeForUser(utcDate: string): string {
  const date = new Date(utcDate);
  return date.toLocaleTimeString(USER_LOCALE, {
    timeZone: USER_TIMEZONE,
    hour: '2-digit',
    minute: '2-digit',
  });
}

/**
 * Formats a UTC date string to show date and time.
 * @param utcDate - ISO 8601 date string in UTC
 * @returns Date and time string (e.g., "31/1/2026, 7:30 p.m.")
 */
export function formatDateTimeForUser(utcDate: string): string {
  const date = new Date(utcDate);
  return date.toLocaleString(USER_LOCALE, {
    timeZone: USER_TIMEZONE,
    year: 'numeric',
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

/**
 * Checks if two UTC dates represent the same day in the user's timezone.
 * @param utcDate1 - First ISO 8601 date string in UTC
 * @param utcDate2 - Second ISO 8601 date string in UTC
 * @returns true if both dates are on the same calendar day in Bogota
 */
export function isSameDay(utcDate1: string, utcDate2: string): boolean {
  const d1 = new Date(utcDate1);
  const d2 = new Date(utcDate2);

  const format = (d: Date) =>
    d.toLocaleDateString(USER_LOCALE, {
      timeZone: USER_TIMEZONE,
      year: 'numeric',
      month: 'numeric',
      day: 'numeric',
    });

  return format(d1) === format(d2);
}

/**
 * Checks if a UTC date represents today in the user's timezone.
 * @param utcDate - ISO 8601 date string in UTC
 * @returns true if the date is today in Bogota
 */
export function isToday(utcDate: string): boolean {
  return isSameDay(utcDate, new Date().toISOString());
}

/**
 * Gets today's date at midnight in Bogota, expressed as a UTC ISO string.
 * This matches the format used in planned_day_utc in the database.
 * @returns ISO 8601 date string representing midnight Bogota in UTC
 */
export function getTodayMidnightUTC(): string {
  const now = new Date();
  // Get current time in Bogota
  const bogotaFormatter = new Intl.DateTimeFormat('en-CA', {
    timeZone: USER_TIMEZONE,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  });
  const bogotaDate = bogotaFormatter.format(now); // "2026-01-31"

  // Parse as midnight in Bogota and convert to UTC
  // Bogota is UTC-5, so midnight Bogota = 05:00 UTC
  return `${bogotaDate}T05:00:00Z`;
}

/**
 * Converts a date string (YYYY-MM-DD) to a UTC timestamp representing
 * midnight of that date in Bogota timezone.
 * @param dateStr - Date string in YYYY-MM-DD format
 * @returns ISO 8601 date string representing midnight Bogota in UTC
 */
export function dateStringToUTC(dateStr: string): string {
  // dateStr is a local Bogota date, midnight = 05:00 UTC
  return `${dateStr}T05:00:00Z`;
}

/**
 * Extracts just the date portion (YYYY-MM-DD) from a UTC timestamp,
 * converting to the user's timezone first.
 * @param utcDate - ISO 8601 date string in UTC
 * @returns Date string in YYYY-MM-DD format (user's local date)
 */
export function getLocalDateString(utcDate: string): string {
  const date = new Date(utcDate);
  const formatter = new Intl.DateTimeFormat('en-CA', {
    timeZone: USER_TIMEZONE,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  });
  return formatter.format(date);
}

/**
 * Gets the weekday name for a UTC date in the user's timezone.
 * @param utcDate - ISO 8601 date string in UTC
 * @returns Weekday name in Spanish (e.g., "viernes")
 */
export function getWeekdayName(utcDate: string): string {
  const date = new Date(utcDate);
  return date.toLocaleDateString(USER_LOCALE, {
    timeZone: USER_TIMEZONE,
    weekday: 'long',
  });
}
