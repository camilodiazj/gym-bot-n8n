#!/usr/bin/env python3
"""
Transform MuscleWiki JSON exercises to Supabase-compatible format.

Usage:
    python transform_exercises.py triceps.json                    # Process single file
    python transform_exercises.py triceps.json --dry-run          # Preview without writing
    python transform_exercises.py --all                           # Process all *.json files
    python transform_exercises.py triceps.json --insert           # Insert directly to Supabase
"""

import json
import csv
import argparse
import os
import re
from pathlib import Path
from typing import Optional
from dataclasses import dataclass, field

# Optional: OpenAI for LLM classification
try:
    from openai import OpenAI
    OPENAI_AVAILABLE = True
except ImportError:
    OPENAI_AVAILABLE = False

# Optional: Supabase for direct insertion
try:
    from supabase import create_client
    SUPABASE_AVAILABLE = True
except ImportError:
    SUPABASE_AVAILABLE = False


# =============================================================================
# CONFIGURATION
# =============================================================================

BASE_URL = "https://musclewiki.com/es-es/exercise/"

# Muscle name mapping: Spanish (JSON) -> English (DB)
MUSCLE_SPANISH_TO_ENGLISH = {
    # Arms
    "Tríceps": "Triceps",
    "Bíceps": "Biceps",
    "Antebrazos": "Forearms",
    # Shoulders (consolidated)
    "Hombros": "Shoulders",
    "Deltoides anterior": "Shoulders",
    "Deltoides lateral": "Shoulders",
    "Deltoides posterior": "Shoulders",
    "Deltoides frontales": "Shoulders",
    "Deltoides posteriores": "Shoulders",
    "Hombros frontales": "Shoulders",
    # Chest
    "Pecho": "Chest",
    "Pectoral superior": "Chest",
    "Pecho medio e inferior": "Chest",
    # Back
    "Espalda": "Back",
    "Dorsales": "Back",
    "Espalda baja": "Lower back",
    # Traps
    "Trapecio": "Traps",
    "Trapecios": "Traps",
    "Trapecios superiores": "Traps",
    "Trapecios inferiores": "Traps (mid-back)",
    "Trapecios Inferiores": "Traps (mid-back)",  # capital I variant
    "Lower Traps": "Traps (mid-back)",
    # Legs
    "Cuádriceps": "Quads",
    "Recto femoral": "Quads",
    "Parte externa del cuádriceps": "Quads",
    "Parte interna del cuádriceps": "Quads",
    "Femorales": "Hamstrings",
    "Isquiotibiales": "Hamstrings",
    "Isquitibiales laterales": "Hamstrings",
    "Isquitibiales mediales": "Hamstrings",
    "Glúteos": "Glutes",
    "Glúteo mayor": "Glutes",
    "Glúteo medio": "Glutes",
    # Lower legs
    "Pantorrillas": "Calfs",
    "Gemelos": "Calfs",
    "Gemelo": "Calfs",
    "Sóleo": "Calfs",
    "Soleo": "Calfs",
    # Core
    "Abdominales": "Abs",
    "Abdominales superiores": "Abs",
    "Abdominales inferiores": "Abs",
    "Oblicuos": "Abs",
    # Other
    "Cuello": "Neck",
    "Pies": "Feet",
    "Ingle": "Groin",
    "Aductores": "Groin",
    "Manos": "Forearms",
    "Extensores de la muñeca": "Forearms",
    "Flexores de la muñeca": "Forearms",
    "Wrist Extensors": "Forearms",
    "Wrist Flexors": "Forearms",
    # English variants from name_en_us (for level 1 fallback)
    "Gluteus Medius": "Glutes",
    "Gluteus Maximus": "Glutes",
    "Lower Abdominals": "Abs",
    "Upper Abdominals": "Abs",
    "Rectus Femoris": "Quads",
    "Vastus Lateralis": "Quads",
    "Vastus Medialis": "Quads",
    "Inner Thigh": "Groin",
    "Hip Adductors": "Groin",
}

# Equipment mapping: JSON category -> DB equipment
EQUIPMENT_MAP = {
    "Dumbbells": "dumbbell",
    "Mancuernas": "dumbbell",
    "Machine": "machine",
    "Máquina": "machine",
    "Cables": "cable",
    "Cable": "cable",
    "Barbell": "barbell",
    "Barra": "barbell",
    "Bodyweight": "bodyweight",
    "Peso corporal": "bodyweight",
    "Smith Machine": "smith",
    "Máquina Smith": "smith",
    "Kettlebells": "kettlebell",
    "Bands": "bands",
    "Bandas": "bands",
}

# Pattern classification by keywords
PATTERN_KEYWORDS = {
    'arm': ['curl', 'extension', 'pushdown', 'kickback', 'skull', 'tricep', 'bicep',
            'hammer', 'preacher', 'concentration', 'wrist'],
    'push_h': ['bench', 'press banca', 'push-up', 'pushup', 'chest press', 'fly',
               'apertura', 'pec deck', 'flexion', 'lagartija'],
    'push_v': ['overhead press', 'shoulder press', 'military', 'dip', 'fondo',
               'pike', 'arnold', 'lateral raise', 'elevacion lateral', 'front raise'],
    'pull_h': ['row', 'remo', 'cable row', 'seated row', 'face pull', 't-bar'],
    'pull_v': ['pulldown', 'pull-up', 'pullup', 'chin-up', 'chinup', 'dominada',
               'lat pulldown', 'jalon', 'pullover'],
    'squat': ['squat', 'sentadilla', 'leg press', 'prensa', 'hack', 'sissy',
              'goblet', 'front squat'],
    'hinge': ['deadlift', 'peso muerto', 'hip thrust', 'romanian', 'rdl',
              'good morning', 'glute bridge', 'puente gluteo', 'hip extension'],
    'lunge': ['lunge', 'zancada', 'split squat', 'step-up', 'step up',
              'bulgarian', 'walking lunge'],
    'core': ['plank', 'plancha', 'crunch', 'abdominal', 'sit-up', 'situp',
             'leg raise', 'elevacion pierna', 'russian twist', 'woodchop',
             'pallof', 'ab wheel', 'hollow', 'dead bug'],
    'accessory': ['calf raise', 'elevacion talon', 'shrug', 'encogimiento',
                  'neck', 'cuello', 'grip', 'wrist curl'],
}

# Pattern to Role mapping
PATTERN_TO_ROLE = {
    'squat': 'compound',
    'hinge': 'compound',
    'lunge': 'compound',
    'push_h': 'compound',
    'push_v': 'compound',
    'pull_h': 'compound',
    'pull_v': 'compound',
    'arm': 'isolation',
    'accessory': 'isolation',
    'core': 'core',
    'carry': 'compound',
    'rotation': 'core',
}

# Keywords that override compound -> isolation
ISOLATION_OVERRIDE_KEYWORDS = [
    'extension', 'curl', 'raise', 'lateral', 'fly', 'apertura',
    'kickback', 'pushdown', 'face pull', 'reverse fly', 'pullover',
    'pec deck', 'cable crossover', 'leg extension', 'leg curl',
]


# =============================================================================
# DEDUPLICATION TRACKING
# =============================================================================

@dataclass
class DeduplicationStats:
    """Track statistics for exercise deduplication across files."""
    total_raw: int = 0
    unique_slugs: int = 0
    duplicates_removed: int = 0
    files_processed: int = 0

    def print_summary(self):
        print(f"\n  Deduplication Summary:")
        print(f"    Files processed: {self.files_processed}")
        print(f"    Total raw exercises: {self.total_raw}")
        print(f"    Unique exercises: {self.unique_slugs}")
        print(f"    Duplicates removed: {self.duplicates_removed}")


# =============================================================================
# HELPER FUNCTIONS
# =============================================================================

def kebab_to_title(s: str) -> str:
    """Convert 'dumbbell-tricep-extension' to 'Dumbbell Tricep Extension'"""
    return ' '.join(word.capitalize() for word in s.split('-'))


def extract_exercise_slug(url_path: str) -> str:
    """Extract the exercise slug from the URL path"""
    return url_path.split('/')[-1]


def extract_canonical_id(url_path: str) -> str:
    """Generate canonical exercise_id from URL slug.

    Example: 'barbell/male/biceps/barbell-curl' -> 'ex_barbell_curl'
    """
    slug = url_path.split('/')[-1]
    return f"ex_{slug.replace('-', '_')}"


def is_unilateral(name: str, url: str) -> str:
    """Detect if the exercise is unilateral"""
    indicators = ['single', 'unilateral', 'one arm', 'one-arm', 'una mano',
                  'un brazo', 'single leg', 'single-leg', 'una pierna']
    combined = (name + ' ' + url).lower()
    return "Yes" if any(ind in combined for ind in indicators) else "No"


def normalize_muscle_name(name: str) -> str:
    """Normalize muscle name for consistent comparison"""
    return name.strip().lower()


def extract_muscles(muscles_list: list) -> tuple[str, str, list[str]]:
    """
    Extract main_muscle and secondary_muscles from the muscles array.

    Returns:
        (main_muscle_en, main_muscle_es, secondary_muscles_es)
    """
    if not muscles_list:
        return "", "", []

    # Filter only level 0 muscles (main muscle groups, not sub-muscles)
    level_0_muscles = [m for m in muscles_list if m.get('level') == 0]

    # If no level 0, fall back to level 1 muscles (sub-muscles)
    if not level_0_muscles:
        level_1_muscles = [m for m in muscles_list if m.get('level') == 1]
        if not level_1_muscles:
            return "", "", []
        # Use the first level 1 muscle
        main = level_1_muscles[0]
        main_muscle_es = main.get('name', '')
        main_muscle_en = MUSCLE_SPANISH_TO_ENGLISH.get(
            main_muscle_es,
            main.get('name_en_us', main_muscle_es)
        )
        secondary = [m.get('name', '') for m in level_1_muscles[1:] if m.get('name')]
        return main_muscle_en, main_muscle_es, secondary

    # First level:0 = main_muscle
    main = level_0_muscles[0]
    main_muscle_es = main.get('name', '')
    main_muscle_en = MUSCLE_SPANISH_TO_ENGLISH.get(
        main_muscle_es,
        main.get('name_en_us', main_muscle_es)
    )

    # Rest of level:0 = secondary_muscles (in Spanish for user-facing queries)
    secondary = [m.get('name', '') for m in level_0_muscles[1:] if m.get('name')]

    return main_muscle_en, main_muscle_es, secondary


def classify_pattern_by_keywords(exercise_name: str, url_slug: str) -> Optional[str]:
    """
    Classify pattern using keywords.
    Returns None if no pattern matches (needs LLM).
    """
    text = (exercise_name + ' ' + url_slug).lower()

    for pattern, keywords in PATTERN_KEYWORDS.items():
        if any(kw in text for kw in keywords):
            return pattern

    return None


def get_role(pattern: str, exercise_name: str) -> str:
    """
    Determine role based on pattern, with isolation overrides.
    """
    if pattern == 'core':
        return 'core'

    name_lower = exercise_name.lower()

    # Check if it's an isolation exercise despite compound pattern
    if any(kw in name_lower for kw in ISOLATION_OVERRIDE_KEYWORDS):
        return 'isolation'

    return PATTERN_TO_ROLE.get(pattern, 'isolation')


def classify_with_llm(exercises: list[dict], api_key: str = None) -> list[dict]:
    """
    Classify exercises using OpenAI LLM.
    Returns list with pattern and role for each exercise.
    """
    if not OPENAI_AVAILABLE:
        print("Warning: OpenAI not available. Using 'accessory' as fallback.")
        return [{'name': ex.get('name', ''), 'pattern': 'accessory', 'role': 'isolation'}
                for ex in exercises]

    api_key = api_key or os.environ.get('OPENAI_API_KEY')
    if not api_key:
        print("Warning: OPENAI_API_KEY not set. Using 'accessory' as fallback.")
        return [{'name': ex.get('name', ''), 'pattern': 'accessory', 'role': 'isolation'}
                for ex in exercises]

    client = OpenAI(api_key=api_key)

    exercise_list = [{'name': ex.get('name', ''), 'spanish_name': ex.get('spanish_name', '')}
                     for ex in exercises]

    prompt = f"""Clasifica cada ejercicio en UN pattern de movimiento y su role.

PATTERNS DISPONIBLES:
- squat: Dominante de rodilla (sentadillas, prensa, extensiones de cuádriceps)
- hinge: Dominante de cadera (peso muerto, hip thrust, curl femoral)
- lunge: Zancadas y movimientos unilaterales de pierna
- push_h: Empuje horizontal (press banca, flexiones, aperturas)
- push_v: Empuje vertical (press militar, elevaciones laterales, fondos)
- pull_h: Tracción horizontal (remos, face pulls)
- pull_v: Tracción vertical (dominadas, jalones, pullovers)
- arm: Aislamiento de brazos (curls, extensiones de tríceps, kickbacks)
- core: Estabilidad abdominal (planchas, crunches, leg raises)
- accessory: Ejercicios complementarios que no encajan en otros patterns

ROLES:
- compound: Multiarticular (usa 2+ articulaciones)
- isolation: Aislamiento (1 articulación)
- core: Ejercicios de core/abdomen

EJERCICIOS A CLASIFICAR:
{json.dumps(exercise_list, ensure_ascii=False, indent=2)}

Responde SOLO con JSON válido, sin markdown ni explicaciones:
[{{"name": "...", "pattern": "...", "role": "..."}}]
"""

    try:
        response = client.chat.completions.create(
            model="gpt-4o-mini",
            messages=[{"role": "user", "content": prompt}],
            temperature=0.1,
        )

        content = response.choices[0].message.content.strip()
        # Clean potential markdown formatting
        if content.startswith('```'):
            content = re.sub(r'^```\w*\n?', '', content)
            content = re.sub(r'\n?```$', '', content)

        return json.loads(content)

    except Exception as e:
        print(f"LLM classification failed: {e}")
        return [{'name': ex.get('name', ''), 'pattern': 'accessory', 'role': 'isolation'}
                for ex in exercises]


def format_secondary_muscles_for_postgres(muscles: list[str]) -> str:
    """Format secondary muscles array for PostgreSQL TEXT[] format"""
    if not muscles:
        return "{}"
    # Escape and format for PostgreSQL array
    escaped = [m.replace('"', '\\"') for m in muscles]
    return "{" + ",".join(f'"{m}"' for m in escaped) + "}"


# =============================================================================
# DEDUPLICATION LOGIC
# =============================================================================

def collect_unique_exercises(json_files: list) -> tuple[dict, DeduplicationStats]:
    """Collect unique exercises across all files by URL slug.

    Returns:
        tuple of (seen_dict, stats) where seen_dict maps slug -> (exercise_data, source_file)
    """
    seen = {}  # slug -> (exercise_data, source_file)
    stats = DeduplicationStats()

    for json_file in json_files:
        try:
            with open(json_file, 'r', encoding='utf-8') as f:
                exercises = json.load(f)
        except (json.JSONDecodeError, IOError) as e:
            print(f"  Warning: Skipping {json_file} - {e}")
            continue

        if not isinstance(exercises, list):
            print(f"  Warning: Skipping {json_file} - not a list")
            continue

        stats.files_processed += 1
        stats.total_raw += len(exercises)

        for ex in exercises:
            url_path = ex.get('target_url', {}).get('male', '')
            if not url_path:
                continue
            slug = extract_exercise_slug(url_path)
            if slug not in seen:
                seen[slug] = (ex, json_file.name if hasattr(json_file, 'name') else str(json_file))
            else:
                stats.duplicates_removed += 1

    stats.unique_slugs = len(seen)
    return seen, stats


# =============================================================================
# MAIN TRANSFORMATION LOGIC
# =============================================================================

def transform_exercise(exercise: dict, index: int = None, muscle_prefix: str = None,
                       pattern_override: str = None, role_override: str = None) -> dict:
    """Transform a single exercise from JSON to DB format"""

    slug = extract_exercise_slug(exercise['target_url']['male'])
    name = exercise.get('name', '')

    # Extract muscles
    main_muscle_en, main_muscle_es, secondary_muscles = extract_muscles(
        exercise.get('muscles', [])
    )

    # Determine pattern
    if pattern_override:
        pattern = pattern_override
    else:
        pattern = classify_pattern_by_keywords(name, slug)
        if not pattern:
            pattern = 'accessory'  # Will be overridden by LLM if available

    # Determine role
    if role_override:
        role = role_override
    else:
        role = get_role(pattern, name)

    # Get equipment
    category_name = exercise.get('category', {}).get('name_en_us', '')
    if not category_name:
        category_name = exercise.get('category', {}).get('name', '')
    equipment = EQUIPMENT_MAP.get(category_name, 'machine')

    # Get level/difficulty
    level = exercise.get('difficulty', {}).get('name', 'Intermedio')

    return {
        'exercise_id': extract_canonical_id(exercise['target_url']['male']),
        'name': kebab_to_title(slug),
        'spanish_name': name,
        'coloquial_name': None,  # NULL to avoid unique constraint violation
        'Músculo Principal': main_muscle_es,
        'pattern': pattern,
        'role': role,
        'main_muscle': main_muscle_en,
        'secondary_muscles': format_secondary_muscles_for_postgres(secondary_muscles),
        'link': BASE_URL + slug,
        'equipment': equipment,
        'unilateral': is_unilateral(name, slug),
        'level': level,
    }


def process_json_file(input_file: str, use_llm: bool = True,
                      openai_key: str = None) -> list[dict]:
    """Process a single JSON file and return transformed exercises"""

    with open(input_file, 'r', encoding='utf-8') as f:
        exercises = json.load(f)

    # Determine muscle prefix from filename
    muscle_prefix = Path(input_file).stem.lower()

    # First pass: classify with keywords
    transformed = []
    needs_llm = []

    for i, ex in enumerate(exercises):
        slug = extract_exercise_slug(ex['target_url']['male'])
        name = ex.get('name', '')
        pattern = classify_pattern_by_keywords(name, slug)

        if pattern:
            transformed.append(transform_exercise(ex, i + 1, muscle_prefix))
        else:
            needs_llm.append({
                'index': i + 1,
                'exercise': ex,
                'name': kebab_to_title(slug),
                'spanish_name': name,
            })

    # Second pass: LLM for unclassified exercises
    if needs_llm and use_llm:
        print(f"  Classifying {len(needs_llm)} exercises with LLM...")
        llm_results = classify_with_llm(needs_llm, openai_key)

        # Create lookup dict
        llm_lookup = {r['name']: r for r in llm_results}

        for item in needs_llm:
            llm_result = llm_lookup.get(item['name'], {})
            pattern = llm_result.get('pattern', 'accessory')
            role = llm_result.get('role', 'isolation')

            transformed.append(transform_exercise(
                item['exercise'],
                item['index'],
                muscle_prefix,
                pattern_override=pattern,
                role_override=role
            ))
    elif needs_llm:
        # No LLM, use accessory as fallback
        for item in needs_llm:
            transformed.append(transform_exercise(
                item['exercise'],
                item['index'],
                muscle_prefix,
                pattern_override='accessory',
                role_override='isolation'
            ))

    # Sort by exercise_id
    transformed.sort(key=lambda x: x['exercise_id'])

    return transformed


def write_csv(exercises: list[dict], output_file: str):
    """Write exercises to CSV file"""
    fieldnames = [
        'exercise_id', 'name', 'spanish_name', 'coloquial_name',
        'Músculo Principal', 'pattern', 'role', 'main_muscle', 'secondary_muscles',
        'link', 'equipment', 'unilateral', 'level'
    ]

    with open(output_file, 'w', newline='', encoding='utf-8') as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(exercises)

    print(f"  Written {len(exercises)} exercises to {output_file}")


def print_preview(exercises: list[dict], limit: int = 10):
    """Print preview of transformed exercises"""
    print("\n  Preview (first {}):\n".format(min(limit, len(exercises))))
    print("  {:15} {:35} {:12} {:10} {:15}".format(
        "ID", "Name", "Pattern", "Role", "Secondary"
    ))
    print("  " + "-" * 90)

    for ex in exercises[:limit]:
        secondary = ex.get('secondary_muscles', '{}')
        if len(secondary) > 15:
            secondary = secondary[:12] + "..."
        print("  {:15} {:35} {:12} {:10} {:15}".format(
            ex['exercise_id'],
            ex['spanish_name'][:35],
            ex['pattern'],
            ex['role'],
            secondary
        ))

    if len(exercises) > limit:
        print(f"\n  ... and {len(exercises) - limit} more exercises")


def insert_to_supabase(exercises: list[dict], supabase_url: str = None,
                       supabase_key: str = None) -> bool:
    """Insert exercises directly to Supabase"""
    if not SUPABASE_AVAILABLE:
        print("Error: Supabase client not available. Install with: pip install supabase")
        return False

    url = supabase_url or os.environ.get('SUPABASE_URL')
    key = supabase_key or os.environ.get('SUPABASE_SERVICE_KEY')

    if not url or not key:
        print("Error: SUPABASE_URL and SUPABASE_SERVICE_KEY must be set")
        return False

    client = create_client(url, key)

    # Convert secondary_muscles from string format to actual array
    for ex in exercises:
        sec = ex.get('secondary_muscles', '{}')
        if isinstance(sec, str):
            # Parse PostgreSQL array format to Python list
            if sec == '{}':
                ex['secondary_muscles'] = []
            else:
                # Remove braces and split
                inner = sec[1:-1]
                ex['secondary_muscles'] = [s.strip('"') for s in inner.split(',') if s]

    try:
        client.table('exercises').upsert(
            exercises,
            on_conflict='exercise_id'
        ).execute()
        print(f"  Inserted/updated {len(exercises)} exercises in Supabase")
        return True
    except Exception as e:
        print(f"Error inserting to Supabase: {e}")
        return False


# =============================================================================
# CLI
# =============================================================================

def main():
    parser = argparse.ArgumentParser(
        description='Transform MuscleWiki JSON exercises to Supabase format',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python transform_exercises.py triceps.json
  python transform_exercises.py triceps.json --dry-run
  python transform_exercises.py --all
  python transform_exercises.py triceps.json --insert
        """
    )

    parser.add_argument('input_file', nargs='?', help='JSON file to process')
    parser.add_argument('--all', action='store_true',
                        help='Process all *.json files in the directory')
    parser.add_argument('--dry-run', action='store_true',
                        help='Preview without writing files')
    parser.add_argument('--insert', action='store_true',
                        help='Insert directly to Supabase')
    parser.add_argument('--output', '-o', help='Output CSV file name')
    parser.add_argument('--no-llm', action='store_true',
                        help='Disable LLM classification (use accessory as fallback)')
    parser.add_argument('--openai-key', help='OpenAI API key (or use OPENAI_API_KEY env)')

    args = parser.parse_args()

    # Determine files to process
    if args.all:
        script_dir = Path(__file__).parent
        raw_dir = script_dir / 'raw'
        if raw_dir.exists():
            json_files = list(raw_dir.glob('*.json'))
        else:
            json_files = list(script_dir.glob('*.json'))
        if not json_files:
            print("No JSON files found in directory")
            return
    elif args.input_file:
        json_files = [Path(args.input_file)]
    else:
        parser.print_help()
        return

    print(f"\nCollecting exercises from {len(json_files)} files...")

    # Pass 1: Collect unique exercises (deduplication)
    unique_exercises, stats = collect_unique_exercises(json_files)

    print(f"  Found {stats.unique_slugs} unique exercises (removed {stats.duplicates_removed} duplicates)")

    # Pass 2: Transform unique exercises
    print(f"\nTransforming exercises...")
    transformed = []
    needs_llm = []

    for slug, (exercise, source_file) in unique_exercises.items():
        name = exercise.get('name', '')
        pattern = classify_pattern_by_keywords(name, slug)

        if pattern:
            transformed.append(transform_exercise(exercise))
        else:
            needs_llm.append({
                'slug': slug,
                'exercise': exercise,
                'name': kebab_to_title(slug),
                'spanish_name': name,
            })

    # LLM classification for unclassified exercises
    if needs_llm and not args.no_llm:
        print(f"  Classifying {len(needs_llm)} exercises with LLM...")
        llm_results = classify_with_llm(needs_llm, args.openai_key)
        llm_lookup = {r['name']: r for r in llm_results}

        for item in needs_llm:
            llm_result = llm_lookup.get(item['name'], {})
            pattern = llm_result.get('pattern', 'accessory')
            role = llm_result.get('role', 'isolation')
            transformed.append(transform_exercise(
                item['exercise'],
                pattern_override=pattern,
                role_override=role
            ))
    elif needs_llm:
        # No LLM, use accessory as fallback
        for item in needs_llm:
            transformed.append(transform_exercise(
                item['exercise'],
                pattern_override='accessory',
                role_override='isolation'
            ))

    # Sort by exercise_id
    transformed.sort(key=lambda x: x['exercise_id'])

    # Output based on flags
    if args.dry_run:
        print_preview(transformed)
        stats.print_summary()
    elif args.insert:
        insert_to_supabase(transformed)
        stats.print_summary()
    else:
        # Write to CSV
        output_file = args.output or 'exercises_all.csv'
        write_csv(transformed, output_file)
        stats.print_summary()

    # Summary
    print(f"\n{'='*60}")
    print(f"Total unique exercises: {len(transformed)}")


if __name__ == "__main__":
    main()
