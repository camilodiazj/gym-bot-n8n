import { describe, it, expect } from 'vitest';
import {
  formatDateForUser,
  formatShortDateForUser,
  formatTimeForUser,
  formatDateTimeForUser,
  isSameDay,
  isToday,
  getTodayMidnightUTC,
  dateStringToUTC,
  getLocalDateString,
  getWeekdayName,
} from './timezone';

describe('Timezone Utilities', () => {
  describe('formatDateForUser', () => {
    it('converts UTC midnight Bogota to correct local date', () => {
      // 2026-01-31 05:00:00 UTC = 2026-01-31 00:00:00 Bogota
      const utcDate = '2026-01-31T05:00:00Z';
      const result = formatDateForUser(utcDate);

      expect(result).toContain('31');
      expect(result.toLowerCase()).toContain('enero');
      expect(result).toContain('2026');
    });

    it('handles late night UTC correctly (previous day in Bogota)', () => {
      // 2026-02-01 00:30:00 UTC = 2026-01-31 19:30:00 Bogota
      const utcDate = '2026-02-01T00:30:00Z';
      const result = formatDateForUser(utcDate);

      // Should show Jan 31, not Feb 1
      expect(result).toContain('31');
      expect(result.toLowerCase()).toContain('enero');
    });

    it('handles early morning UTC correctly (same day in Bogota)', () => {
      // 2026-02-01 05:00:00 UTC = 2026-02-01 00:00:00 Bogota
      const utcDate = '2026-02-01T05:00:00Z';
      const result = formatDateForUser(utcDate);

      // Should show Feb 1
      expect(result).toContain('1');
      expect(result.toLowerCase()).toContain('febrero');
    });
  });

  describe('formatShortDateForUser', () => {
    it('returns short date format', () => {
      const utcDate = '2026-01-31T05:00:00Z';
      const result = formatShortDateForUser(utcDate);

      // Format should be like "31/1/2026" or similar
      expect(result).toMatch(/31.*1.*2026|2026.*1.*31/);
    });
  });

  describe('formatTimeForUser', () => {
    it('converts UTC to Bogota time', () => {
      // 2026-02-01 00:30:00 UTC = 7:30 PM Bogota (19:30)
      const utcDate = '2026-02-01T00:30:00Z';
      const result = formatTimeForUser(utcDate);

      // Should show 7:30 PM in some format
      expect(result).toMatch(/7:30|19:30/);
    });

    it('handles midnight UTC (7 PM previous day in Bogota)', () => {
      // 2026-01-31 00:00:00 UTC = 2026-01-30 19:00:00 Bogota
      const utcDate = '2026-01-31T00:00:00Z';
      const result = formatTimeForUser(utcDate);

      expect(result).toMatch(/7:00|19:00/);
    });

    it('handles 05:00 UTC (midnight Bogota)', () => {
      // 2026-01-31 05:00:00 UTC = 00:00 Bogota
      const utcDate = '2026-01-31T05:00:00Z';
      const result = formatTimeForUser(utcDate);

      expect(result).toMatch(/12:00|0:00|00:00/);
    });
  });

  describe('formatDateTimeForUser', () => {
    it('shows both date and time', () => {
      const utcDate = '2026-02-01T00:30:00Z';
      const result = formatDateTimeForUser(utcDate);

      // Should contain date (31/1/2026) and time (7:30)
      expect(result).toContain('31');
      expect(result).toMatch(/7:30|19:30/);
    });
  });

  describe('isSameDay', () => {
    it('returns true for same Bogota day', () => {
      // Both are Jan 31 in Bogota
      const date1 = '2026-01-31T05:00:00Z'; // midnight Bogota
      const date2 = '2026-02-01T04:59:00Z'; // 11:59 PM Bogota same day

      expect(isSameDay(date1, date2)).toBe(true);
    });

    it('returns false for different Bogota days', () => {
      const date1 = '2026-01-31T05:00:00Z'; // Jan 31 midnight Bogota
      const date2 = '2026-02-01T05:00:00Z'; // Feb 1 midnight Bogota

      expect(isSameDay(date1, date2)).toBe(false);
    });

    it('handles timezone boundary correctly', () => {
      // 4:59 AM UTC Feb 1 = 11:59 PM Jan 31 Bogota
      // 5:00 AM UTC Feb 1 = 00:00 Feb 1 Bogota
      const beforeMidnight = '2026-02-01T04:59:00Z';
      const afterMidnight = '2026-02-01T05:00:00Z';

      expect(isSameDay(beforeMidnight, afterMidnight)).toBe(false);
    });

    it('same UTC day but different Bogota days at boundary', () => {
      // Both are Feb 1 in UTC, but one is Jan 31 in Bogota
      const jan31Evening = '2026-02-01T02:00:00Z'; // Jan 31, 9 PM Bogota
      const feb1Morning = '2026-02-01T10:00:00Z'; // Feb 1, 5 AM Bogota

      expect(isSameDay(jan31Evening, feb1Morning)).toBe(false);
    });
  });

  describe('dateStringToUTC', () => {
    it('converts date string to UTC midnight Bogota', () => {
      const result = dateStringToUTC('2026-01-31');

      expect(result).toBe('2026-01-31T05:00:00Z');
    });

    it('handles different dates correctly', () => {
      expect(dateStringToUTC('2026-02-01')).toBe('2026-02-01T05:00:00Z');
      expect(dateStringToUTC('2026-12-31')).toBe('2026-12-31T05:00:00Z');
    });
  });

  describe('getLocalDateString', () => {
    it('extracts local date from UTC timestamp', () => {
      // 2026-02-01 00:30:00 UTC = 2026-01-31 19:30:00 Bogota
      const utcDate = '2026-02-01T00:30:00Z';
      const result = getLocalDateString(utcDate);

      expect(result).toBe('2026-01-31');
    });

    it('handles midnight Bogota correctly', () => {
      // 2026-01-31 05:00:00 UTC = 2026-01-31 00:00:00 Bogota
      const utcDate = '2026-01-31T05:00:00Z';
      const result = getLocalDateString(utcDate);

      expect(result).toBe('2026-01-31');
    });
  });

  describe('getWeekdayName', () => {
    it('returns weekday in Spanish', () => {
      // Jan 31, 2026 is a Saturday
      const utcDate = '2026-01-31T05:00:00Z';
      const result = getWeekdayName(utcDate);

      expect(result.toLowerCase()).toBe('sábado');
    });

    it('handles cross-timezone weekday correctly', () => {
      // 2026-02-01 00:30:00 UTC = Jan 31 in Bogota (Saturday)
      const utcDate = '2026-02-01T00:30:00Z';
      const result = getWeekdayName(utcDate);

      expect(result.toLowerCase()).toBe('sábado');
    });
  });
});

describe('Edge Cases', () => {
  it('handles year boundary correctly', () => {
    // Dec 31 Bogota at 11:59 PM = Jan 1 UTC at 04:59
    const dec31 = '2027-01-01T04:59:00Z';
    const result = formatDateForUser(dec31);

    expect(result).toContain('31');
    expect(result.toLowerCase()).toContain('diciembre');
    expect(result).toContain('2026');
  });

  it('handles first day of year', () => {
    // Jan 1 Bogota at 00:00 = Jan 1 UTC at 05:00
    const jan1 = '2026-01-01T05:00:00Z';
    const result = formatDateForUser(jan1);

    expect(result).toContain('1');
    expect(result.toLowerCase()).toContain('enero');
    expect(result).toContain('2026');
  });

  it('Colombia has no DST - offset is always -5', () => {
    // Both summer and winter should show same offset behavior
    const summer = '2026-07-15T05:00:00Z';
    const winter = '2026-01-15T05:00:00Z';

    // Both should show midnight (00:00) when converted to local time
    const summerResult = formatTimeForUser(summer);
    const winterResult = formatTimeForUser(winter);

    expect(summerResult).toMatch(/12:00|0:00|00:00/);
    expect(winterResult).toMatch(/12:00|0:00|00:00/);
  });

  it('handles leap year correctly', () => {
    // Feb 29, 2028 (leap year)
    const leapDay = '2028-02-29T05:00:00Z';
    const result = formatDateForUser(leapDay);

    expect(result).toContain('29');
    expect(result.toLowerCase()).toContain('febrero');
    expect(result).toContain('2028');
  });
});

describe('Planned Day UTC Verification', () => {
  it('planned_day_utc should always have hour = 5 (midnight Bogota)', () => {
    // These are valid planned_day_utc values as stored in the database
    const plannedDays = [
      '2026-01-31T05:00:00Z',
      '2026-02-01T05:00:00Z',
      '2026-12-31T05:00:00Z',
    ];

    for (const pd of plannedDays) {
      const date = new Date(pd);
      expect(date.getUTCHours()).toBe(5);
      expect(date.getUTCMinutes()).toBe(0);
      expect(date.getUTCSeconds()).toBe(0);
    }
  });

  it('getLocalDateString preserves the correct date', () => {
    // For all valid planned_day_utc values, the local date should match
    const testCases = [
      { utc: '2026-01-31T05:00:00Z', localDate: '2026-01-31' },
      { utc: '2026-02-01T05:00:00Z', localDate: '2026-02-01' },
      { utc: '2026-12-31T05:00:00Z', localDate: '2026-12-31' },
    ];

    for (const { utc, localDate } of testCases) {
      expect(getLocalDateString(utc)).toBe(localDate);
    }
  });
});
