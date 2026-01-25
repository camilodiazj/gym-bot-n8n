#!/usr/bin/env python3
"""
Unit tests for transform_exercises.py

Run with:
    pytest test_transform_exercises.py -v
    python -m pytest test_transform_exercises.py -v
"""

import pytest
import json
import tempfile
from pathlib import Path
from transform_exercises import (
    kebab_to_title,
    extract_exercise_slug,
    extract_canonical_id,
    is_unilateral,
    normalize_muscle_name,
    extract_muscles,
    classify_pattern_by_keywords,
    get_role,
    format_secondary_muscles_for_postgres,
    collect_unique_exercises,
    transform_exercise,
    DeduplicationStats,
    MUSCLE_SPANISH_TO_ENGLISH,
)


# =============================================================================
# HELPER FUNCTION TESTS
# =============================================================================

class TestKebabToTitle:
    """Test kebab_to_title function"""

    def test_basic_conversion(self):
        assert kebab_to_title("barbell-curl") == "Barbell Curl"
        assert kebab_to_title("dumbbell-lateral-raise") == "Dumbbell Lateral Raise"

    def test_single_word(self):
        assert kebab_to_title("plank") == "Plank"

    def test_empty_string(self):
        assert kebab_to_title("") == ""

    def test_multiple_dashes(self):
        assert kebab_to_title("one-two-three-four") == "One Two Three Four"


class TestExtractExerciseSlug:
    """Test extract_exercise_slug function"""

    def test_standard_path(self):
        assert extract_exercise_slug("barbell/male/biceps/barbell-curl") == "barbell-curl"

    def test_short_path(self):
        assert extract_exercise_slug("barbell-curl") == "barbell-curl"

    def test_path_with_trailing_slash(self):
        assert extract_exercise_slug("barbell/male/biceps/") == ""

    def test_complex_slug(self):
        assert extract_exercise_slug("kettlebell/male/glutes/kettlebell-single-arm-curtsy-lunge") == \
               "kettlebell-single-arm-curtsy-lunge"


class TestExtractCanonicalId:
    """Test extract_canonical_id function"""

    def test_basic_id_generation(self):
        assert extract_canonical_id("barbell/male/biceps/barbell-curl") == "ex_barbell_curl"

    def test_complex_id(self):
        assert extract_canonical_id("kettlebell/male/glutes/kettlebell-single-arm-curtsy-lunge") == \
               "ex_kettlebell_single_arm_curtsy_lunge"

    def test_consistency(self):
        # Same slug should always produce same ID
        url1 = "barbell/male/biceps/barbell-curl"
        url2 = "barbell/female/biceps/barbell-curl"
        assert extract_canonical_id(url1) == extract_canonical_id(url2)

    def test_slug_with_numbers(self):
        assert extract_canonical_id("path/to/exercise-123") == "ex_exercise_123"


class TestIsUnilateral:
    """Test is_unilateral function"""

    def test_single_arm_detection(self):
        assert is_unilateral("Single Arm Row", "single-arm-row") == "Yes"
        assert is_unilateral("One arm curl", "one-arm-curl") == "Yes"

    def test_single_leg_detection(self):
        assert is_unilateral("Single leg squat", "single-leg-squat") == "Yes"
        assert is_unilateral("Una pierna zancada", "lunge") == "Yes"

    def test_bilateral_exercises(self):
        assert is_unilateral("Barbell curl", "barbell-curl") == "No"
        assert is_unilateral("Bench press", "bench-press") == "No"

    def test_spanish_keywords(self):
        assert is_unilateral("Curl con una mano", "curl") == "Yes"
        assert is_unilateral("Press con un brazo", "press") == "Yes"


class TestNormalizeMucleName:
    """Test normalize_muscle_name function"""

    def test_lowercase_conversion(self):
        assert normalize_muscle_name("Biceps") == "biceps"
        assert normalize_muscle_name("TRICEPS") == "triceps"

    def test_strip_whitespace(self):
        assert normalize_muscle_name("  Shoulders  ") == "shoulders"
        assert normalize_muscle_name("\tChest\n") == "chest"

    def test_empty_string(self):
        assert normalize_muscle_name("") == ""


class TestExtractMuscles:
    """Test extract_muscles function"""

    def test_single_muscle(self):
        muscles = [
            {"name": "Tríceps", "name_en_us": "Triceps", "level": 0}
        ]
        main_en, main_es, secondary = extract_muscles(muscles)
        assert main_en == "Triceps"
        assert main_es == "Tríceps"
        assert secondary == []

    def test_multiple_level_0_muscles(self):
        muscles = [
            {"name": "Tríceps", "name_en_us": "Triceps", "level": 0},
            {"name": "Pecho", "name_en_us": "Chest", "level": 0}
        ]
        main_en, main_es, secondary = extract_muscles(muscles)
        assert main_en == "Triceps"
        assert main_es == "Tríceps"
        assert secondary == ["Pecho"]

    def test_ignores_level_1_muscles(self):
        muscles = [
            {"name": "Tríceps", "name_en_us": "Triceps", "level": 0},
            {"name": "Cabeza larga", "level": 1}
        ]
        main_en, main_es, secondary = extract_muscles(muscles)
        assert main_en == "Triceps"
        assert main_es == "Tríceps"
        assert secondary == []

    def test_multiple_secondaries(self):
        muscles = [
            {"name": "Tríceps", "level": 0},
            {"name": "Pecho", "level": 0},
            {"name": "Hombros", "level": 0},
            {"name": "Cabeza larga", "level": 1}
        ]
        main_en, main_es, secondary = extract_muscles(muscles)
        assert main_en == "Triceps"
        assert secondary == ["Pecho", "Hombros"]

    def test_empty_list(self):
        main_en, main_es, secondary = extract_muscles([])
        assert main_en == ""
        assert main_es == ""
        assert secondary == []

    def test_no_level_0_muscles(self):
        muscles = [
            {"name": "Cabeza larga", "level": 1},
            {"name": "Cabeza medial", "level": 1}
        ]
        main_en, main_es, secondary = extract_muscles(muscles)
        assert main_en == ""
        assert main_es == ""
        assert secondary == []

    def test_muscle_mapping_fallback(self):
        # Test with muscle not in MUSCLE_SPANISH_TO_ENGLISH
        muscles = [
            {"name": "Unknown Muscle", "name_en_us": "Unknown", "level": 0}
        ]
        main_en, main_es, secondary = extract_muscles(muscles)
        assert main_en == "Unknown"
        assert main_es == "Unknown Muscle"


class TestClassifyPatternByKeywords:
    """Test classify_pattern_by_keywords function"""

    def test_arm_pattern(self):
        assert classify_pattern_by_keywords("Barbell Curl", "barbell-curl") == "arm"
        assert classify_pattern_by_keywords("Tricep Extension", "tricep-extension") == "arm"
        assert classify_pattern_by_keywords("Hammer Curl", "hammer-curl") == "arm"

    def test_push_h_pattern(self):
        assert classify_pattern_by_keywords("Bench Press", "bench-press") == "push_h"
        assert classify_pattern_by_keywords("Push-up", "push-up") == "push_h"
        assert classify_pattern_by_keywords("Dumbbell Fly", "dumbbell-fly") == "push_h"

    def test_push_v_pattern(self):
        assert classify_pattern_by_keywords("Overhead Press", "overhead-press") == "push_v"
        assert classify_pattern_by_keywords("Lateral Raise", "lateral-raise") == "push_v"
        assert classify_pattern_by_keywords("Arnold Press", "arnold-press") == "push_v"

    def test_pull_h_pattern(self):
        assert classify_pattern_by_keywords("Seated Row", "seated-row") == "pull_h"
        assert classify_pattern_by_keywords("Face Pull", "face-pull") == "pull_h"

    def test_pull_v_pattern(self):
        assert classify_pattern_by_keywords("Pull-up", "pull-up") == "pull_v"
        assert classify_pattern_by_keywords("Lat Pulldown", "lat-pulldown") == "pull_v"

    def test_squat_pattern(self):
        assert classify_pattern_by_keywords("Squat", "squat") == "squat"
        assert classify_pattern_by_keywords("Leg Press", "leg-press") == "squat"

    def test_hinge_pattern(self):
        assert classify_pattern_by_keywords("Deadlift", "deadlift") == "hinge"
        assert classify_pattern_by_keywords("Hip Thrust", "hip-thrust") == "hinge"
        assert classify_pattern_by_keywords("Romanian Deadlift", "romanian-deadlift") == "hinge"

    def test_lunge_pattern(self):
        assert classify_pattern_by_keywords("Walking Lunge", "walking-lunge") == "lunge"
        assert classify_pattern_by_keywords("Split Squat", "split-squat") == "lunge"

    def test_core_pattern(self):
        assert classify_pattern_by_keywords("Plank", "plank") == "core"
        assert classify_pattern_by_keywords("Crunch", "crunch") == "core"
        assert classify_pattern_by_keywords("Leg Raise", "leg-raise") == "core"

    def test_accessory_pattern(self):
        assert classify_pattern_by_keywords("Calf Raise", "calf-raise") == "accessory"
        assert classify_pattern_by_keywords("Shrug", "shrug") == "accessory"

    def test_no_match_returns_none(self):
        assert classify_pattern_by_keywords("Unknown Exercise", "unknown-exercise") is None

    def test_spanish_keywords(self):
        assert classify_pattern_by_keywords("Press banca", "press-banca") == "push_h"
        assert classify_pattern_by_keywords("Sentadilla", "sentadilla") == "squat"
        assert classify_pattern_by_keywords("Peso muerto", "peso-muerto") == "hinge"


class TestGetRole:
    """Test get_role function"""

    def test_core_pattern_returns_core(self):
        assert get_role("core", "Plank") == "core"

    def test_compound_patterns(self):
        assert get_role("squat", "Barbell Squat") == "compound"
        assert get_role("hinge", "Deadlift") == "compound"
        assert get_role("lunge", "Walking Lunge") == "compound"

    def test_isolation_patterns(self):
        assert get_role("arm", "Bicep Curl") == "isolation"
        assert get_role("accessory", "Calf Raise") == "isolation"

    def test_isolation_override(self):
        # Even if pattern suggests compound, isolation keywords override
        assert get_role("push_v", "Lateral Raise") == "isolation"
        assert get_role("push_h", "Dumbbell Fly") == "isolation"
        assert get_role("pull_v", "Pullover") == "isolation"

    def test_default_isolation(self):
        assert get_role("unknown_pattern", "Unknown Exercise") == "isolation"


class TestFormatSecondaryMusclesForPostgres:
    """Test format_secondary_muscles_for_postgres function"""

    def test_empty_list(self):
        assert format_secondary_muscles_for_postgres([]) == "{}"

    def test_single_muscle(self):
        assert format_secondary_muscles_for_postgres(["Pecho"]) == '{"Pecho"}'

    def test_multiple_muscles(self):
        result = format_secondary_muscles_for_postgres(["Pecho", "Hombros", "Tríceps"])
        assert result == '{"Pecho","Hombros","Tríceps"}'

    def test_escapes_quotes(self):
        result = format_secondary_muscles_for_postgres(['Test "Quote"'])
        assert result == '{"Test \\"Quote\\""}'

    def test_preserves_spaces(self):
        result = format_secondary_muscles_for_postgres(["Glúteo mayor", "Glúteo medio"])
        assert result == '{"Glúteo mayor","Glúteo medio"}'


# =============================================================================
# DEDUPLICATION TESTS
# =============================================================================

class TestDeduplicationStats:
    """Test DeduplicationStats dataclass"""

    def test_initialization(self):
        stats = DeduplicationStats()
        assert stats.total_raw == 0
        assert stats.unique_slugs == 0
        assert stats.duplicates_removed == 0
        assert stats.files_processed == 0

    def test_manual_values(self):
        stats = DeduplicationStats(
            total_raw=100,
            unique_slugs=80,
            duplicates_removed=20,
            files_processed=5
        )
        assert stats.total_raw == 100
        assert stats.unique_slugs == 80
        assert stats.duplicates_removed == 20
        assert stats.files_processed == 5

    def test_print_summary(self, capsys):
        stats = DeduplicationStats(
            total_raw=3471,
            unique_slugs=1616,
            duplicates_removed=1855,
            files_processed=43
        )
        stats.print_summary()
        captured = capsys.readouterr()
        assert "Files processed: 43" in captured.out
        assert "Total raw exercises: 3471" in captured.out
        assert "Unique exercises: 1616" in captured.out
        assert "Duplicates removed: 1855" in captured.out


class TestCollectUniqueExercises:
    """Test collect_unique_exercises function"""

    def test_single_file_no_duplicates(self, tmp_path):
        # Create temporary JSON file
        exercises = [
            {
                "name": "Curl con barra",
                "target_url": {"male": "barbell/male/biceps/barbell-curl"},
                "muscles": [{"name": "Bíceps", "level": 0}],
                "category": {"name_en_us": "Barbell"},
                "difficulty": {"name": "Intermedio"}
            },
            {
                "name": "Press de banca",
                "target_url": {"male": "barbell/male/chest/bench-press"},
                "muscles": [{"name": "Pecho", "level": 0}],
                "category": {"name_en_us": "Barbell"},
                "difficulty": {"name": "Intermedio"}
            }
        ]
        json_file = tmp_path / "test.json"
        json_file.write_text(json.dumps(exercises))

        unique, stats = collect_unique_exercises([json_file])

        assert stats.files_processed == 1
        assert stats.total_raw == 2
        assert stats.unique_slugs == 2
        assert stats.duplicates_removed == 0
        assert len(unique) == 2
        assert "barbell-curl" in unique
        assert "bench-press" in unique

    def test_multiple_files_with_duplicates(self, tmp_path):
        # File 1
        exercises1 = [
            {
                "name": "Curl con barra",
                "target_url": {"male": "barbell/male/biceps/barbell-curl"},
                "muscles": [{"name": "Bíceps", "level": 0}],
                "category": {"name_en_us": "Barbell"},
                "difficulty": {"name": "Intermedio"}
            }
        ]
        file1 = tmp_path / "biceps.json"
        file1.write_text(json.dumps(exercises1))

        # File 2 (duplicate curl)
        exercises2 = [
            {
                "name": "Curl con barra",
                "target_url": {"male": "barbell/male/biceps/barbell-curl"},
                "muscles": [{"name": "Bíceps", "level": 0}],
                "category": {"name_en_us": "Barbell"},
                "difficulty": {"name": "Intermedio"}
            },
            {
                "name": "Press de banca",
                "target_url": {"male": "barbell/male/chest/bench-press"},
                "muscles": [{"name": "Pecho", "level": 0}],
                "category": {"name_en_us": "Barbell"},
                "difficulty": {"name": "Intermedio"}
            }
        ]
        file2 = tmp_path / "chest.json"
        file2.write_text(json.dumps(exercises2))

        unique, stats = collect_unique_exercises([file1, file2])

        assert stats.files_processed == 2
        assert stats.total_raw == 3
        assert stats.unique_slugs == 2
        assert stats.duplicates_removed == 1
        assert len(unique) == 2

    def test_invalid_json_file(self, tmp_path, capsys):
        # Create invalid JSON file
        invalid_file = tmp_path / "invalid.json"
        invalid_file.write_text("not valid json {")

        unique, stats = collect_unique_exercises([invalid_file])

        captured = capsys.readouterr()
        assert "Warning: Skipping" in captured.out
        assert stats.files_processed == 0
        assert len(unique) == 0

    def test_non_list_json(self, tmp_path, capsys):
        # JSON is valid but not a list
        json_file = tmp_path / "dict.json"
        json_file.write_text('{"key": "value"}')

        unique, stats = collect_unique_exercises([json_file])

        captured = capsys.readouterr()
        assert "not a list" in captured.out
        assert stats.files_processed == 0

    def test_exercise_without_url(self, tmp_path):
        # Exercise missing target_url
        exercises = [
            {
                "name": "Exercise 1",
                "target_url": {"male": "path/to/exercise-1"}
            },
            {
                "name": "Exercise 2",
                # Missing target_url
            }
        ]
        json_file = tmp_path / "test.json"
        json_file.write_text(json.dumps(exercises))

        unique, stats = collect_unique_exercises([json_file])

        assert stats.unique_slugs == 1  # Only one valid
        assert len(unique) == 1


# =============================================================================
# TRANSFORMATION TESTS
# =============================================================================

class TestTransformExercise:
    """Test transform_exercise function"""

    def test_basic_transformation(self):
        exercise = {
            "name": "Curl con barra",
            "target_url": {"male": "barbell/male/biceps/barbell-curl"},
            "muscles": [{"name": "Bíceps", "name_en_us": "Biceps", "level": 0}],
            "category": {"name_en_us": "Barbell"},
            "difficulty": {"name": "Intermedio"}
        }

        result = transform_exercise(exercise)

        assert result['exercise_id'] == "ex_barbell_curl"
        assert result['name'] == "Barbell Curl"
        assert result['spanish_name'] == "Curl con barra"
        assert result['main_muscle'] == "Biceps"
        assert result['Músculo Principal'] == "Bíceps"
        assert result['equipment'] == "barbell"
        assert result['level'] == "Intermedio"
        assert result['link'] == "https://musclewiki.com/es-es/exercise/barbell-curl"
        assert result['pattern'] == "arm"
        assert result['role'] == "isolation"

    def test_exercise_with_secondary_muscles(self):
        exercise = {
            "name": "Press de banca",
            "target_url": {"male": "barbell/male/chest/bench-press"},
            "muscles": [
                {"name": "Pecho", "level": 0},
                {"name": "Tríceps", "level": 0},
                {"name": "Hombros", "level": 0}
            ],
            "category": {"name_en_us": "Barbell"},
            "difficulty": {"name": "Intermedio"}
        }

        result = transform_exercise(exercise)

        assert result['main_muscle'] == "Chest"
        assert result['secondary_muscles'] == '{"Tríceps","Hombros"}'

    def test_pattern_override(self):
        exercise = {
            "name": "Unknown Exercise",
            "target_url": {"male": "path/to/unknown-exercise"},
            "muscles": [{"name": "Bíceps", "level": 0}],
            "category": {"name_en_us": "Barbell"},
            "difficulty": {"name": "Principiante"}
        }

        result = transform_exercise(exercise, pattern_override="squat")

        assert result['pattern'] == "squat"
        assert result['role'] == "compound"

    def test_role_override(self):
        exercise = {
            "name": "Test Exercise",
            "target_url": {"male": "path/to/test-exercise"},
            "muscles": [{"name": "Bíceps", "level": 0}],
            "category": {"name_en_us": "Barbell"},
            "difficulty": {"name": "Avanzado"}
        }

        result = transform_exercise(exercise, role_override="core")

        assert result['role'] == "core"

    def test_unilateral_detection(self):
        exercise = {
            "name": "Curl con una mano",
            "target_url": {"male": "dumbbell/male/biceps/single-arm-curl"},
            "muscles": [{"name": "Bíceps", "level": 0}],
            "category": {"name_en_us": "Dumbbells"},
            "difficulty": {"name": "Intermedio"}
        }

        result = transform_exercise(exercise)

        assert result['unilateral'] == "Yes"

    def test_equipment_mapping(self):
        test_cases = [
            ("Dumbbells", "dumbbell"),
            ("Mancuernas", "dumbbell"),
            ("Barbell", "barbell"),
            ("Machine", "machine"),
            ("Cables", "cable"),
            ("Bodyweight", "bodyweight"),
        ]

        for category, expected_equipment in test_cases:
            exercise = {
                "name": "Test",
                "target_url": {"male": "path/to/test"},
                "muscles": [{"name": "Bíceps", "level": 0}],
                "category": {"name_en_us": category},
                "difficulty": {"name": "Intermedio"}
            }
            result = transform_exercise(exercise)
            assert result['equipment'] == expected_equipment

    def test_coloquial_name_is_null(self):
        exercise = {
            "name": "Test",
            "target_url": {"male": "path/to/test"},
            "muscles": [{"name": "Bíceps", "level": 0}],
            "category": {"name_en_us": "Barbell"},
            "difficulty": {"name": "Intermedio"}
        }

        result = transform_exercise(exercise)

        assert result['coloquial_name'] is None


# =============================================================================
# INTEGRATION TESTS
# =============================================================================

class TestEndToEndDeduplication:
    """Integration tests for the full deduplication pipeline"""

    def test_realistic_scenario(self, tmp_path):
        # Simulate the real scenario: same exercise in multiple muscle files
        barbell_curl_data = {
            "name": "Curl con barra",
            "target_url": {"male": "barbell/male/biceps/barbell-curl"},
            "muscles": [{"name": "Bíceps", "level": 0}],
            "category": {"name_en_us": "Barbell"},
            "difficulty": {"name": "Intermedio"}
        }

        bench_press_data = {
            "name": "Press de banca",
            "target_url": {"male": "barbell/male/chest/bench-press"},
            "muscles": [
                {"name": "Pecho", "level": 0},
                {"name": "Tríceps", "level": 0}
            ],
            "category": {"name_en_us": "Barbell"},
            "difficulty": {"name": "Intermedio"}
        }

        # File 1: Biceps (has curl)
        file1 = tmp_path / "biceps.json"
        file1.write_text(json.dumps([barbell_curl_data]))

        # File 2: Triceps (has bench press - also works triceps)
        file2 = tmp_path / "triceps.json"
        file2.write_text(json.dumps([bench_press_data]))

        # File 3: Arms (has both - duplicates!)
        file3 = tmp_path / "arms.json"
        file3.write_text(json.dumps([barbell_curl_data, bench_press_data]))

        # Collect unique
        unique, stats = collect_unique_exercises([file1, file2, file3])

        assert stats.files_processed == 3
        assert stats.total_raw == 4
        assert stats.unique_slugs == 2
        assert stats.duplicates_removed == 2

        # Transform unique exercises
        transformed = []
        for slug, (exercise, source_file) in unique.items():
            transformed.append(transform_exercise(exercise))

        assert len(transformed) == 2

        # Check IDs are consistent
        ids = {ex['exercise_id'] for ex in transformed}
        assert "ex_barbell_curl" in ids
        assert "ex_bench_press" in ids


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
