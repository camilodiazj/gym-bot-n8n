# 02 - HTML Template Spec: GenerateRoutineHTML Code Node

**Role:** pixel-dev (developer)
**Status:** Implementation Guide
**Last Updated:** 2026-02-06

---

## 1. Node Configuration

| Property | Value |
|----------|-------|
| **Type** | `n8n-nodes-base.code` |
| **TypeVersion** | 2 |
| **Name** | `GenerateRoutineHTML` |
| **Position** | `[1760, 400]` |
| **executeOnce** | `true` |
| **Language** | JavaScript |

This node sits after `GetWeek1WithExercises` (Postgres query) and before the email-send node. It transforms raw exercise rows and user profile data into a single HTML string suitable for email delivery.

---

## 2. Input Data Contract

### 2.1 Exercise Data: `$input.all()`

Source: Output of the `GetWeek1WithExercises` Postgres node. Each item in the array represents one exercise row.

```typescript
interface ExerciseRow {
  day_name: string;              // "Push", "Pull", "Legs", "Upper", "Lower", etc.
  exercise_order: number;        // 1-based sequential within day
  sets: string;                  // "3" or "2-3"
  reps: string;                  // "10-12" or "10"
  rir: string;                   // "3" or "2-3"
  rest_seconds: number;          // 60, 90, 120, 180
  tempo: string;                 // "2-0-2-0"
  spanish_name: string;          // "Press de Banca con Barra"
  main_muscle: string;           // "Chest" (English from DB)
  secondary_muscles: string[] | null; // ["Triceps", "Shoulders"] or null
  equipment: string;             // "barbell", "dumbbell", "bodyweight", etc.
  link: string | null;           // MuscleWiki URL or null
  role: string;                  // "compound", "core", "isolation"
}
```

### 2.2 User Profile: `$('ProcessUserPreferences').first().json`

Source: Output of the `ProcessUserPreferences` Code node earlier in the workflow.

```typescript
interface UserProfile {
  full_name: string;
  email: string;
  primary_goal: string;           // "Ganar masa muscular", "Bajar grasa", etc.
  fitness_level: string;          // "Principiante", "Intermedio", "Avanzado"
  days_available: number;         // 3, 4, 5, 6
  priority_muscles: string;       // "Gluteo, pierna" or null
  biological_sex: string;         // "M" or "F"
  session_duration_mins: string;  // "45-60 minutos", "60-75 minutos", etc.
  processed: {
    environment: string;          // "GYM" or "HOME"
    home: {
      is_home: boolean;
      equipment_list: string[];   // ["bodyweight", "dumbbell", "resistance_band"]
      equipment_tier: string;     // "minimal", "basic", "intermediate", "advanced"
    }
  }
}
```

---

## 3. Output Data Contract

The node must return a **single item** (array of one) with this shape:

```typescript
interface OutputItem {
  html: string;       // Complete HTML email body (inline CSS, no external dependencies)
  email: string;      // User's email address (pass-through from profile)
  subject: string;    // "Tu Rutina Semana 1 - {fullName} | Kairos"
  fullName: string;   // User's full name (pass-through from profile)
}
```

**Return statement:**

```javascript
return [{ json: { html, email, subject, fullName } }];
```

---

## 4. Complete JavaScript Code

The following is the **full, production-ready** JavaScript code for the `jsCode` field of the n8n Code node. Copy-paste directly.

```javascript
// =============================================================================
// GenerateRoutineHTML - n8n Code Node (JavaScript, typeVersion 2)
// =============================================================================
// Transforms exercise data + user profile into an email-safe HTML string.
// All CSS is inline. No external resources. Table-based layout.
// =============================================================================

// ---------------------------------------------------------------------------
// 1. TRANSLATION MAPS
// ---------------------------------------------------------------------------

const equipmentMap = {
  'barbell': 'Barra',
  'dumbbell': 'Mancuerna',
  'bodyweight': 'Peso corporal',
  'machine': 'Maquina',
  'cable': 'Cable',
  'resistance_band': 'Banda elastica',
  'kettlebell': 'Kettlebell',
  'ez_bar': 'Barra EZ',
  'smith_machine': 'Maquina Smith',
  'pull_bar': 'Barra de dominadas',
  'bench': 'Banco',
  'trap_bar': 'Trap Bar',
  'bands': 'Bandas',
};

const muscleMap = {
  'Chest': 'Pecho',
  'Back': 'Espalda',
  'Shoulders': 'Hombros',
  'Biceps': 'Biceps',
  'Triceps': 'Triceps',
  'Quads': 'Cuadriceps',
  'Hamstrings': 'Isquiotibiales',
  'Glutes': 'Gluteos',
  'Calfs': 'Pantorrillas',
  'Abs': 'Abdominales',
  'Core': 'Core',
  'Forearms': 'Antebrazos',
  'Traps': 'Trapecios',
  'Lower back': 'Espalda baja',
  'Lats': 'Dorsales',
  'Hip Flexors': 'Flexores de cadera',
  'Adductors': 'Aductores',
  'Abductors': 'Abductores',
  'Obliques': 'Oblicuos',
  'Neck': 'Cuello',
  'Serratus Anterior': 'Serrato Anterior',
};

// ---------------------------------------------------------------------------
// 2. HELPER FUNCTIONS
// ---------------------------------------------------------------------------

/**
 * Translate equipment DB value (English) to Spanish display string.
 * Falls back to the raw value with first letter capitalized.
 */
function translateEquipment(eq) {
  if (!eq) return '-';
  return equipmentMap[eq] || eq.charAt(0).toUpperCase() + eq.slice(1).replace(/_/g, ' ');
}

/**
 * Translate muscle name from English DB value to Spanish.
 * Falls back to the raw value.
 */
function translateMuscle(muscle) {
  if (!muscle) return '-';
  return muscleMap[muscle] || muscle;
}

/**
 * Format rest_seconds into a human-readable string.
 * >= 120 seconds shows minutes, otherwise shows seconds with "s" suffix.
 */
function formatRest(seconds) {
  if (!seconds && seconds !== 0) return '-';
  if (seconds >= 120) return `${seconds / 60} min`;
  return `${seconds}s`;
}

/**
 * Return an emoji string based on the day_name pattern.
 */
function getDayEmoji(dayName) {
  const lower = dayName.toLowerCase();
  if (lower.includes('push')) return '\u{1F3CB}\u{FE0F}';
  if (lower.includes('pull')) return '\u{1F4AA}';
  if (lower.includes('leg') || lower.includes('lower')) return '\u{1F9B5}';
  if (lower.includes('upper')) return '\u{1F9BE}';
  if (lower.includes('full')) return '\u{26A1}';
  if (lower.includes('glute')) return '\u{1F351}';
  return '\u{1F525}';
}

/**
 * Return the Spanish label for an exercise role.
 */
function getRoleLabel(role) {
  const labels = {
    'compound': 'Ejercicios Compuestos',
    'core': 'Core',
    'isolation': 'Ejercicios de Aislamiento',
  };
  return labels[role] || role;
}

/**
 * Given an array of exercises for a day, extract the unique main muscles,
 * translate them to Spanish, and join them into a readable string.
 * Uses "y" before the last muscle (Spanish grammar).
 */
function getDayMuscles(exercises) {
  const seen = new Set();
  const muscles = [];
  for (const ex of exercises) {
    const translated = translateMuscle(ex.main_muscle);
    if (!seen.has(translated)) {
      seen.add(translated);
      muscles.push(translated);
    }
  }
  if (muscles.length === 0) return '';
  if (muscles.length === 1) return muscles[0];
  return muscles.slice(0, -1).join(', ') + ' y ' + muscles[muscles.length - 1];
}

/**
 * Translate the user's home equipment list to Spanish for display.
 */
function translateEquipmentList(list) {
  if (!list || list.length === 0) return 'Peso corporal';
  return list.map(e => translateEquipment(e)).join(', ');
}

// ---------------------------------------------------------------------------
// 3. DATA LOADING
// ---------------------------------------------------------------------------

const userProfile = $('ProcessUserPreferences').first().json;
const exercises = $input.all().map(item => item.json);

const fullName = userProfile.full_name || 'Atleta';
const email = userProfile.email || '';
const primaryGoal = userProfile.primary_goal || '-';
const fitnessLevel = userProfile.fitness_level || '-';
const daysAvailable = userProfile.days_available || '-';
const priorityMuscles = userProfile.priority_muscles || null;
const environment = userProfile.processed?.environment || 'GYM';
const isHome = userProfile.processed?.home?.is_home || false;
const homeEquipmentList = userProfile.processed?.home?.equipment_list || [];

const subject = `Tu Rutina Semana 1 - ${fullName} | Kairos`;

// ---------------------------------------------------------------------------
// 4. GROUP EXERCISES BY DAY (preserving insertion order)
// ---------------------------------------------------------------------------

const dayMap = new Map();
for (const ex of exercises) {
  const dayName = ex.day_name;
  if (!dayMap.has(dayName)) {
    dayMap.set(dayName, []);
  }
  dayMap.get(dayName).push(ex);
}

// Sort exercises within each day by exercise_order
for (const [, dayExercises] of dayMap) {
  dayExercises.sort((a, b) => a.exercise_order - b.exercise_order);
}

// ---------------------------------------------------------------------------
// 5. INLINE CSS CONSTANTS
// ---------------------------------------------------------------------------

const CSS = {
  body: 'font-family: Arial, Helvetica, sans-serif; max-width: 700px; margin: 0 auto; background-color: #f8f9fa; padding: 20px; color: #212529; line-height: 1.6;',
  header: 'background: #1a1a2e; color: #ffffff; padding: 30px 20px; text-align: center; border-radius: 12px 12px 0 0;',
  headerH1: 'margin: 0 0 8px 0; font-size: 24px; font-weight: 700; color: #ffffff;',
  headerSub: 'margin: 0; font-size: 16px; color: #d1d5db; font-weight: 400;',
  content: 'background: #ffffff; padding: 24px 20px; border: 1px solid #e0e0e0; border-top: none; border-radius: 0 0 12px 12px;',
  sectionTitle: 'font-size: 20px; font-weight: 700; color: #1a1a2e; margin: 28px 0 12px 0; padding-bottom: 8px; border-bottom: 3px solid #e63946;',
  dayTitle: 'font-size: 20px; font-weight: 700; color: #1a1a2e; margin: 32px 0 4px 0; padding-bottom: 8px; border-bottom: 3px solid #e63946;',
  daySubtitle: 'margin: 0 0 16px 0; font-size: 14px; color: #6c757d; font-style: italic;',
  roleHeader: 'font-size: 14px; font-weight: 700; color: #495057; margin: 16px 0 8px 0; padding: 8px 12px; background-color: #f1f3f5; border-radius: 6px; border-left: 4px solid #e63946;',
  table: 'width: 100%; border-collapse: collapse; margin-bottom: 16px; font-size: 13px;',
  th: 'background-color: #f1f3f5; padding: 10px 8px; text-align: left; font-weight: 700; color: #495057; border: 1px solid #e9ecef; font-size: 12px; text-transform: uppercase; letter-spacing: 0.5px;',
  td: 'padding: 8px; border: 1px solid #e9ecef; vertical-align: top; color: #212529;',
  tdCenter: 'padding: 8px; border: 1px solid #e9ecef; vertical-align: top; color: #212529; text-align: center;',
  link: 'color: #e63946; text-decoration: none; font-weight: 600;',
  profileTable: 'width: 100%; border-collapse: collapse; margin-bottom: 20px;',
  profileLabel: 'padding: 8px 12px; font-weight: 700; color: #495057; background-color: #f8f9fa; border: 1px solid #e9ecef; width: 40%;',
  profileValue: 'padding: 8px 12px; color: #212529; border: 1px solid #e9ecef;',
  tipBox: 'background-color: #fff3cd; border-left: 4px solid #ffc107; padding: 12px 16px; margin: 12px 0 20px 0; font-size: 13px; color: #664d03; border-radius: 0 6px 6px 0;',
  noteH3: 'font-size: 15px; font-weight: 700; color: #1a1a2e; margin: 16px 0 8px 0;',
  noteUl: 'margin: 0 0 12px 0; padding-left: 20px; font-size: 13px; color: #495057;',
  noteLi: 'margin-bottom: 4px;',
  quoteBox: 'background-color: #f8f9fa; border-left: 4px solid #e63946; padding: 16px 20px; margin: 24px 0; font-style: italic; color: #495057; font-size: 14px; border-radius: 0 6px 6px 0;',
  footer: 'text-align: center; color: #868e96; font-size: 12px; margin-top: 24px; padding-top: 16px; border-top: 1px solid #e9ecef;',
  hr: 'border: none; border-top: 1px solid #e9ecef; margin: 24px 0;',
  overviewTd: 'padding: 8px 12px; border: 1px solid #e9ecef; color: #212529;',
  overviewTdBold: 'padding: 8px 12px; border: 1px solid #e9ecef; color: #212529; font-weight: 700;',
};

// ---------------------------------------------------------------------------
// 6. HTML GENERATION FUNCTIONS
// ---------------------------------------------------------------------------

/**
 * Build the profile summary table rows.
 */
function buildProfileSection() {
  let rows = '';
  rows += `<tr><td style="${CSS.profileLabel}">Objetivo</td><td style="${CSS.profileValue}">${primaryGoal}</td></tr>`;
  rows += `<tr><td style="${CSS.profileLabel}">Nivel</td><td style="${CSS.profileValue}">${fitnessLevel}</td></tr>`;
  rows += `<tr><td style="${CSS.profileLabel}">Dias/semana</td><td style="${CSS.profileValue}">${daysAvailable} dias</td></tr>`;
  rows += `<tr><td style="${CSS.profileLabel}">Ambiente</td><td style="${CSS.profileValue}">${environment === 'HOME' ? 'Casa (HOME)' : 'Gimnasio (GYM)'}</td></tr>`;

  if (environment === 'HOME') {
    const eqDisplay = translateEquipmentList(homeEquipmentList);
    rows += `<tr><td style="${CSS.profileLabel}">Equipo disponible</td><td style="${CSS.profileValue}">${eqDisplay}</td></tr>`;
  } else {
    rows += `<tr><td style="${CSS.profileLabel}">Equipo</td><td style="${CSS.profileValue}">Gimnasio completo</td></tr>`;
  }

  if (priorityMuscles) {
    rows += `<tr><td style="${CSS.profileLabel}">Musculos prioritarios</td><td style="${CSS.profileValue}">${priorityMuscles}</td></tr>`;
  }

  return `<table style="${CSS.profileTable}">${rows}</table>`;
}

/**
 * Build the Quick Reference Guide table.
 */
function buildQuickReference() {
  return `
    <h2 style="${CSS.sectionTitle}">Guia Rapida</h2>
    <table style="${CSS.table}">
      <tr>
        <th style="${CSS.th}">Termino</th>
        <th style="${CSS.th}">Significado</th>
      </tr>
      <tr>
        <td style="${CSS.td}"><strong>Sets</strong></td>
        <td style="${CSS.td}">Numero de series a realizar</td>
      </tr>
      <tr>
        <td style="${CSS.td}"><strong>Reps</strong></td>
        <td style="${CSS.td}">Repeticiones por serie</td>
      </tr>
      <tr>
        <td style="${CSS.td}"><strong>RIR</strong></td>
        <td style="${CSS.td}">Repeticiones en reserva (cuantas mas podrias hacer)</td>
      </tr>
      <tr>
        <td style="${CSS.td}"><strong>Descanso</strong></td>
        <td style="${CSS.td}">Tiempo de pausa entre series</td>
      </tr>
    </table>
    <div style="${CSS.tipBox}">
      <strong>Tip:</strong> RIR 3 significa que al terminar cada serie, deberias sentir que podrias hacer 3 repeticiones mas. No llegues al fallo muscular.
    </div>
  `;
}

/**
 * Build the Weekly Overview table.
 */
function buildWeeklyOverview() {
  let rows = '';
  let dayIndex = 1;
  for (const [dayName, dayExercises] of dayMap) {
    const muscles = getDayMuscles(dayExercises);
    rows += `
      <tr>
        <td style="${CSS.overviewTd}">${dayIndex}</td>
        <td style="${CSS.overviewTdBold}">${dayName}</td>
        <td style="${CSS.overviewTd}">${muscles}</td>
      </tr>
    `;
    dayIndex++;
  }

  return `
    <h2 style="${CSS.sectionTitle}">Resumen de la Semana</h2>
    <table style="${CSS.table}">
      <tr>
        <th style="${CSS.th}">Dia</th>
        <th style="${CSS.th}">Sesion</th>
        <th style="${CSS.th}">Enfoque Principal</th>
      </tr>
      ${rows}
    </table>
  `;
}

/**
 * Build a single exercise table for a role group.
 * @param {Array} roleExercises - exercises filtered to one role
 * @param {string} role - "compound", "core", or "isolation"
 */
function buildRoleTable(roleExercises, role) {
  if (!roleExercises || roleExercises.length === 0) return '';

  // Use the first exercise's set parameters as the group header values
  const first = roleExercises[0];
  const setsDisplay = first.sets || '-';
  const repsDisplay = first.reps || '-';
  const rirDisplay = first.rir || '-';
  const restDisplay = formatRest(first.rest_seconds);
  const roleLabel = getRoleLabel(role);

  const headerText = `${roleLabel} (${setsDisplay} sets x ${repsDisplay} reps | RIR ${rirDisplay} | ${restDisplay} descanso)`;

  let exerciseRows = '';
  for (const ex of roleExercises) {
    const musclePrimary = translateMuscle(ex.main_muscle);
    const secondaryArr = ex.secondary_muscles || [];
    const secondaryStr = secondaryArr.map(m => translateMuscle(m)).join(', ');
    const muscleDisplay = secondaryStr ? `${musclePrimary}, ${secondaryStr}` : musclePrimary;
    const equipDisplay = translateEquipment(ex.equipment);
    const linkCell = ex.link
      ? `<a href="${ex.link}" style="${CSS.link}" target="_blank">Ver tecnica</a>`
      : '-';

    exerciseRows += `
      <tr>
        <td style="${CSS.tdCenter}">${ex.exercise_order}</td>
        <td style="${CSS.td}">${ex.spanish_name}</td>
        <td style="${CSS.td}">${muscleDisplay}</td>
        <td style="${CSS.td}">${equipDisplay}</td>
        <td style="${CSS.tdCenter}">${linkCell}</td>
      </tr>
    `;
  }

  return `
    <div style="${CSS.roleHeader}">${headerText}</div>
    <table style="${CSS.table}">
      <tr>
        <th style="${CSS.th}; width: 30px;">#</th>
        <th style="${CSS.th}">Ejercicio</th>
        <th style="${CSS.th}">Musculo Principal</th>
        <th style="${CSS.th}">Equipo</th>
        <th style="${CSS.th}; width: 90px;">Video</th>
      </tr>
      ${exerciseRows}
    </table>
  `;
}

/**
 * Build a complete day section with all role groups.
 */
function buildDaySection(dayName, dayExercises) {
  const emoji = getDayEmoji(dayName);
  const muscles = getDayMuscles(dayExercises);

  // Sub-group by role in defined order: compound -> core -> isolation
  const roleOrder = ['compound', 'core', 'isolation'];
  const byRole = {};
  for (const ex of dayExercises) {
    const role = ex.role || 'compound';
    if (!byRole[role]) byRole[role] = [];
    byRole[role].push(ex);
  }

  let roleSections = '';
  for (const role of roleOrder) {
    if (byRole[role] && byRole[role].length > 0) {
      roleSections += buildRoleTable(byRole[role], role);
    }
  }

  // Handle any roles not in the predefined order
  for (const role of Object.keys(byRole)) {
    if (!roleOrder.includes(role)) {
      roleSections += buildRoleTable(byRole[role], role);
    }
  }

  return `
    <h2 style="${CSS.dayTitle}">${emoji} ${dayName}</h2>
    <p style="${CSS.daySubtitle}">${muscles}</p>
    ${roleSections}
    <hr style="${CSS.hr}" />
  `;
}

/**
 * Build the Notes section.
 */
function buildNotesSection() {
  return `
    <h2 style="${CSS.sectionTitle}">Notas Importantes</h2>

    <h3 style="${CSS.noteH3}">Calentamiento (5-10 min antes de cada sesion)</h3>
    <ul style="${CSS.noteUl}">
      <li style="${CSS.noteLi}">Movilidad articular general</li>
      <li style="${CSS.noteLi}">5 minutos de cardio ligero (caminar, saltar suave)</li>
      <li style="${CSS.noteLi}">1 serie ligera del primer ejercicio</li>
    </ul>

    <h3 style="${CSS.noteH3}">Progresion</h3>
    <ul style="${CSS.noteUl}">
      <li style="${CSS.noteLi}"><strong>Semana 1-2:</strong> Enfocate en aprender la tecnica correcta</li>
      <li style="${CSS.noteLi}"><strong>Semana 3:</strong> Semana de descarga (menos volumen)</li>
      <li style="${CSS.noteLi}"><strong>Semana 4:</strong> Intensificacion</li>
    </ul>

    <h3 style="${CSS.noteH3}">Hidratacion y Nutricion</h3>
    <ul style="${CSS.noteUl}">
      <li style="${CSS.noteLi}">Bebe agua antes, durante y despues del entrenamiento</li>
      <li style="${CSS.noteLi}">Come algo ligero 1-2 horas antes de entrenar</li>
      <li style="${CSS.noteLi}">Proteina despues del entrenamiento para recuperacion</li>
    </ul>
  `;
}

/**
 * Build the motivational quote block.
 */
function buildMotivationalQuote() {
  return `
    <div style="${CSS.quoteBox}">
      "No se trata de ser perfecto, se trata de ser constante. Cada repeticion cuenta, cada dia que te presentas es una victoria."
    </div>
  `;
}

// ---------------------------------------------------------------------------
// 7. ASSEMBLE FULL HTML
// ---------------------------------------------------------------------------

let daySections = '';
for (const [dayName, dayExercises] of dayMap) {
  daySections += buildDaySection(dayName, dayExercises);
}

const html = `<!DOCTYPE html>
<html lang="es">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>${subject}</title>
</head>
<body style="${CSS.body}">
  <div style="max-width: 700px; margin: 0 auto;">

    <!-- HEADER -->
    <div style="${CSS.header}">
      <h1 style="${CSS.headerH1}">Plan de Entrenamiento - Semana 1</h1>
      <p style="${CSS.headerSub}">${fullName}</p>
    </div>

    <!-- CONTENT BODY -->
    <div style="${CSS.content}">

      <!-- Profile Summary -->
      ${buildProfileSection()}

      <!-- Quick Reference Guide -->
      ${buildQuickReference()}

      <!-- Weekly Overview -->
      ${buildWeeklyOverview()}

      <!-- Horizontal Rule -->
      <hr style="${CSS.hr}" />

      <!-- Per-Day Sections -->
      ${daySections}

      <!-- Notes Section -->
      ${buildNotesSection()}

      <!-- Motivational Quote -->
      ${buildMotivationalQuote()}

      <!-- Footer -->
      <p style="${CSS.footer}">
        Generado por Kairos Personal Trainer
      </p>

    </div>

  </div>
</body>
</html>`;

// ---------------------------------------------------------------------------
// 8. RETURN OUTPUT
// ---------------------------------------------------------------------------

return [{ json: { html, email, subject, fullName } }];
```

---

## 5. Edge Cases

| Case | Behavior |
|------|----------|
| **No exercises for a role** (e.g., no core exercises in a Push day) | The `buildRoleTable` function returns an empty string; the sub-section is completely omitted |
| **`secondary_muscles` is null** | The muscle display shows only the translated `main_muscle`, no trailing comma |
| **`link` is null** | The Video column shows `"-"` instead of a hyperlink |
| **HOME environment** | Profile summary shows "Casa (HOME)" for Ambiente and lists translated equipment |
| **GYM environment** | Profile summary shows "Gimnasio (GYM)" for Ambiente and "Gimnasio completo" for Equipo |
| **`priority_muscles` is null or empty** | The "Musculos prioritarios" row is omitted entirely from the profile table |
| **Sets/reps are ranges** (e.g., `"2-3"`, `"10-12"`) | Displayed as-is in the role group header and not parsed further |
| **Unknown equipment value** | `translateEquipment` falls back to capitalizing the first letter and replacing underscores with spaces |
| **Unknown muscle value** | `translateMuscle` falls back to the raw English value |
| **Unknown role value** | `getRoleLabel` falls back to the raw role string; `buildDaySection` still renders it after the three known roles |
| **`rest_seconds` is 0 or missing** | `formatRest` returns `"0s"` for 0, `"-"` for undefined/null |
| **No exercises at all** | The day sections area is empty; the rest of the email still renders correctly |
| **Long day names** | No truncation is applied; HTML table cells will wrap text naturally |
| **User profile fields missing** | Defaults are applied: `fullName` defaults to `"Atleta"`, other fields default to `"-"` |

---

## 6. HTML/CSS Rules for Email Compatibility

These rules are **already enforced** in the code above. They exist here as a checklist for future modifications.

### Mandatory Rules

| Rule | Rationale |
|------|-----------|
| **All CSS must be inline** (`style="..."`) | Gmail, Outlook, Yahoo strip `<style>` and `<link>` tags |
| **Table-based layout** for structure | Flexbox/Grid not supported in Outlook, older Gmail |
| **Max width 700px**, centered via `margin: 0 auto` | Standard email viewport; prevents horizontal scrolling |
| **Font: Arial, Helvetica, sans-serif** | Web-safe stack supported by all email clients |
| **No JavaScript** | Stripped by 100% of email clients |
| **Colors as hex values** only | CSS variables, `hsl()`, `oklch()` not supported |
| **Explicit color on links** | Some clients override link colors; inline style prevents this |
| **No external images** | Text-only email for maximum deliverability |
| **No `<div>` nesting > 3 levels** | Deep nesting can break in Outlook Word renderer |
| **`border-collapse: collapse`** on all tables | Prevents double borders in Outlook |

### Color Palette

| Element | Hex | Usage |
|---------|-----|-------|
| Header background | `#1a1a2e` | Dark navy header block |
| Header text | `#ffffff` | White text on dark header |
| Header subtitle | `#d1d5db` | Muted white for subtitle |
| Accent color | `#e63946` | Section borders, links, role header left border |
| Table header bg | `#f1f3f5` | Light gray for `<th>` rows and role group headers |
| Table borders | `#e9ecef` | Lighter gray for cell borders |
| Body text | `#212529` | Near-black for readability |
| Muted text | `#495057` | Role headers, secondary text |
| Subtle text | `#6c757d` | Day subtitles |
| Footer text | `#868e96` | Gray footer |
| Tip box bg | `#fff3cd` | Yellow background for tip callout |
| Tip box border | `#ffc107` | Amber left border for tip callout |
| Tip box text | `#664d03` | Dark amber text in tip callout |
| Quote box bg | `#f8f9fa` | Light gray background for motivational quote |
| Page background | `#f8f9fa` | Light gray body background |
| Content background | `#ffffff` | White content area |

---

## 7. Equipment Translation Map (Complete)

Reference for all known `equipment` values in the `exercises` table and their Spanish translations:

```javascript
const equipmentMap = {
  'barbell': 'Barra',
  'dumbbell': 'Mancuerna',
  'bodyweight': 'Peso corporal',
  'machine': 'Maquina',
  'cable': 'Cable',
  'resistance_band': 'Banda elastica',
  'kettlebell': 'Kettlebell',
  'ez_bar': 'Barra EZ',
  'smith_machine': 'Maquina Smith',
  'pull_bar': 'Barra de dominadas',
  'bench': 'Banco',
  'trap_bar': 'Trap Bar',
  'bands': 'Bandas',
};
```

If a new equipment type is added to the database and not found in this map, the `translateEquipment` function will capitalize the first letter and replace underscores with spaces (e.g., `"leg_press"` becomes `"Leg press"`).

---

## 8. Muscle Translation Map (Complete)

Reference for all known `main_muscle` and `secondary_muscles` values in the `exercises` table:

```javascript
const muscleMap = {
  'Chest': 'Pecho',
  'Back': 'Espalda',
  'Shoulders': 'Hombros',
  'Biceps': 'Biceps',
  'Triceps': 'Triceps',
  'Quads': 'Cuadriceps',
  'Hamstrings': 'Isquiotibiales',
  'Glutes': 'Gluteos',
  'Calfs': 'Pantorrillas',
  'Abs': 'Abdominales',
  'Core': 'Core',
  'Forearms': 'Antebrazos',
  'Traps': 'Trapecios',
  'Lower back': 'Espalda baja',
  'Lats': 'Dorsales',
  'Hip Flexors': 'Flexores de cadera',
  'Adductors': 'Aductores',
  'Abductors': 'Abductores',
  'Obliques': 'Oblicuos',
  'Neck': 'Cuello',
  'Serratus Anterior': 'Serrato Anterior',
};
```

If a new muscle is added to the database and not found in this map, the `translateMuscle` function returns the raw English value as-is.

---

## 9. Testing the Code Node

### 9.1 In-n8n Testing (Recommended First Step)

1. **Pin test data** on the `GetWeek1WithExercises` node with at least 3 days of sample exercises covering all three roles (compound, core, isolation).
2. **Pin test data** on `ProcessUserPreferences` with a complete user profile (test both GYM and HOME environments).
3. **Execute** the `GenerateRoutineHTML` node.
4. **Inspect** the output: confirm `html`, `email`, `subject`, and `fullName` fields are present.
5. **Copy** the `html` value, paste into a `.html` file, and open in a browser.

### 9.2 Sample Pin Data for GetWeek1WithExercises

```json
[
  {
    "day_name": "Push",
    "exercise_order": 1,
    "sets": "3",
    "reps": "10-12",
    "rir": "3",
    "rest_seconds": 90,
    "tempo": "2-0-2-0",
    "spanish_name": "Press de Banca con Barra",
    "main_muscle": "Chest",
    "secondary_muscles": ["Triceps", "Shoulders"],
    "equipment": "barbell",
    "link": "https://musclewiki.com/es-es/exercise/barbell-bench-press",
    "role": "compound"
  },
  {
    "day_name": "Push",
    "exercise_order": 2,
    "sets": "3",
    "reps": "10-12",
    "rir": "3",
    "rest_seconds": 90,
    "tempo": "2-0-2-0",
    "spanish_name": "Press Militar con Barra",
    "main_muscle": "Shoulders",
    "secondary_muscles": ["Triceps"],
    "equipment": "barbell",
    "link": "https://musclewiki.com/es-es/exercise/barbell-overhead-press",
    "role": "compound"
  },
  {
    "day_name": "Push",
    "exercise_order": 3,
    "sets": "2",
    "reps": "12-15",
    "rir": "3",
    "rest_seconds": 60,
    "tempo": "2-0-2-0",
    "spanish_name": "Aperturas con Mancuerna",
    "main_muscle": "Chest",
    "secondary_muscles": null,
    "equipment": "dumbbell",
    "link": null,
    "role": "isolation"
  },
  {
    "day_name": "Pull",
    "exercise_order": 1,
    "sets": "3",
    "reps": "8-10",
    "rir": "3",
    "rest_seconds": 120,
    "tempo": "2-0-2-0",
    "spanish_name": "Remo con Barra",
    "main_muscle": "Back",
    "secondary_muscles": ["Biceps"],
    "equipment": "barbell",
    "link": "https://musclewiki.com/es-es/exercise/barbell-row",
    "role": "compound"
  },
  {
    "day_name": "Pull",
    "exercise_order": 2,
    "sets": "2",
    "reps": "10-15",
    "rir": "3",
    "rest_seconds": 60,
    "tempo": "2-0-2-0",
    "spanish_name": "Dead Bug",
    "main_muscle": "Abs",
    "secondary_muscles": null,
    "equipment": "bodyweight",
    "link": "https://musclewiki.com/es-es/exercise/dead-bug",
    "role": "core"
  }
]
```

### 9.3 Sample Pin Data for ProcessUserPreferences

```json
{
  "full_name": "Xiomara Alejandra Diaz Ramirez",
  "email": "xiomara@example.com",
  "primary_goal": "Salud general / recomposicion corporal",
  "fitness_level": "Principiante",
  "days_available": 5,
  "priority_muscles": "Gluteo, pierna",
  "biological_sex": "F",
  "session_duration_mins": "45-60 minutos",
  "processed": {
    "environment": "HOME",
    "home": {
      "is_home": true,
      "equipment_list": ["bodyweight", "dumbbell", "resistance_band"],
      "equipment_tier": "basic"
    }
  }
}
```

### 9.4 Email Client Testing

After browser validation, send the HTML via email and verify rendering in:

| Client | Priority | Known Issues |
|--------|----------|--------------|
| **Gmail (Web)** | High | Strips `<style>` tags; inline CSS required |
| **Gmail (Mobile)** | High | May reduce font sizes; test readability |
| **Outlook (Desktop)** | Medium | Uses Word renderer; tables render well, `border-radius` ignored |
| **Apple Mail** | Medium | Generally good HTML support |
| **Yahoo Mail** | Low | Similar to Gmail in behavior |

### 9.5 Validation Checklist

- [ ] All 5+ day sections render with correct emoji
- [ ] Compound/Core/Isolation sub-sections appear only when exercises exist for that role
- [ ] Exercise order numbers are correct within each day
- [ ] Equipment names display in Spanish
- [ ] Muscle names display in Spanish
- [ ] Video links are clickable and open in new tab
- [ ] Missing links show "-" instead of broken anchor
- [ ] Profile summary shows correct environment (GYM vs HOME)
- [ ] HOME environment shows equipment list in Spanish
- [ ] Priority muscles row appears only when provided
- [ ] Quick Reference table renders correctly
- [ ] Weekly Overview table lists all days with muscle focus
- [ ] Notes section has all three sub-sections
- [ ] Motivational quote renders in styled blockquote
- [ ] Footer shows "Generado por Kairos Personal Trainer"
- [ ] Subject line matches format: "Tu Rutina Semana 1 - {name} | Kairos"
- [ ] Email renders at 700px max width, centered
- [ ] No horizontal scrolling on mobile (320px viewport)

---

## 10. Node Integration Diagram

```
                                          +---------------------------+
                                          |  ProcessUserPreferences   |
                                          |  (Code node - earlier)    |
                                          +-------------|-------------+
                                                        |
                                                        | $('ProcessUserPreferences').first().json
                                                        |
+-------------------------+     $input.all()     +------v------------------+     html, email,     +-----------------+
| GetWeek1WithExercises   | ------------------> | GenerateRoutineHTML     | --- subject, -----> | Send Email      |
| (Postgres query)        |                      | (THIS Code node)        |     fullName         | (Email node)    |
+-------------------------+                      +-------------------------+                     +-----------------+
```

### Upstream SQL Query (GetWeek1WithExercises)

For reference, the Postgres node queries:

```sql
SELECT
  w.day_name,
  w.exercise_order,
  w.sets,
  w.reps,
  w.rir,
  w."rest-seconds" AS rest_seconds,
  w.tempo,
  e.spanish_name,
  e.main_muscle,
  e.secondary_muscles,
  e.equipment,
  e.link,
  e.role
FROM workouts w
JOIN exercises e ON w.exercise_id = e.exercise_id
WHERE w.user_id = $1
  AND w.week = 1
ORDER BY w.day_name, w.exercise_order;
```

**Note:** The `day_name` ordering from SQL depends on insertion order. The `Map` in JavaScript preserves insertion order, so the days appear in the same sequence as the query results. If deterministic day ordering is needed, add an explicit `ORDER BY` on a day-number column.

### Downstream Email Node

The email-send node (Gmail or SMTP) maps these output fields:

| Email Node Field | Source |
|-----------------|--------|
| To | `{{ $json.email }}` |
| Subject | `{{ $json.subject }}` |
| HTML Body | `{{ $json.html }}` |

---

## 11. Maintenance Notes

### Adding a New Equipment Type

1. Add the English key and Spanish translation to `equipmentMap` in the Code node.
2. No other changes needed; the fallback handles unknown values gracefully.

### Adding a New Muscle

1. Add the English key and Spanish translation to `muscleMap` in the Code node.
2. No other changes needed.

### Adding a New Exercise Role

1. If the role should appear in a specific position (e.g., between compound and isolation), add it to the `roleOrder` array in `buildDaySection`.
2. Add a Spanish label to the `labels` object inside `getRoleLabel`.

### Modifying the Color Scheme

1. Update the hex values in the `CSS` object at the top of the code.
2. All colors are centralized there; no need to search through the HTML template.

### Adding Accent Characters (Tildes)

The current code intentionally omits accent characters (tildes) in the HTML output for maximum email client compatibility. If accented characters are desired (e.g., "Musculos" -> "Musculos"), update the string literals throughout the code. The translation maps and static strings would need updating. Note that UTF-8 encoding is declared in the `<meta charset="UTF-8" />` tag, so accented characters are technically supported.
