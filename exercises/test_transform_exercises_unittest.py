#!/usr/bin/env python3
"""
Unit tests for transform_exercises.py using unittest (no external dependencies)

Run with:
    python3 test_transform_exercises_unittest.py
    python3 -m unittest test_transform_exercises_unittest -v
"""

import unittest
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
)


class TestKebabToTitle(unittest.TestCase):
    """Test kebab_to_title function"""

    def test_basic_conversion(self):
        self.assertEqual(kebab_to_title("barbell-curl"), "Barbell Curl")
        self.assertEqual(kebab_to_title("dumbbell-lateral-raise"), "Dumbbell Lateral Raise")

    def test_single_word(self):
        self.assertEqual(kebab_to_title("plank"), "Plank")

    def test_empty_string(self):
        self.assertEqual(kebab_to_title(""), "")


class TestExtractExerciseSlug(unittest.TestCase):
    """Test extract_exercise_slug function"""

    def test_standard_path(self):
        self.assertEqual(extract_exercise_slug("barbell/male/biceps/barbell-curl"), "barbell-curl")

    def test_complex_slug(self):
        result = extract_exercise_slug("kettlebell/male/glutes/kettlebell-single-arm-curtsy-lunge")
        self.assertEqual(result, "kettlebell-single-arm-curtsy-lunge")


class TestExtractCanonicalId(unittest.TestCase):
    """Test extract_canonical_id function"""

    def test_basic_id_generation(self):
        self.assertEqual(extract_canonical_id("barbell/male/biceps/barbell-curl"), "ex_barbell_curl")

    def test_consistency(self):
        # Same slug should always produce same ID
        url1 = "barbell/male/biceps/barbell-curl"
        url2 = "barbell/female/biceps/barbell-curl"
        self.assertEqual(extract_canonical_id(url1), extract_canonical_id(url2))


class TestIsUnilateral(unittest.TestCase):
    """Test is_unilateral function"""

    def test_single_arm_detection(self):
        self.assertEqual(is_unilateral("Single Arm Row", "single-arm-row"), "Yes")
        self.assertEqual(is_unilateral("One arm curl", "one-arm-curl"), "Yes")

    def test_bilateral_exercises(self):
        self.assertEqual(is_unilateral("Barbell curl", "barbell-curl"), "No")
        self.assertEqual(is_unilateral("Bench press", "bench-press"), "No")


class TestExtractMuscles(unittest.TestCase):
    """Test extract_muscles function"""

    def test_single_muscle(self):
        muscles = [
            {"name": "Tríceps", "name_en_us": "Triceps", "level": 0}
        ]
        main_en, main_es, secondary = extract_muscles(muscles)
        self.assertEqual(main_en, "Triceps")
        self.assertEqual(main_es, "Tríceps")
        self.assertEqual(secondary, [])

    def test_multiple_level_0_muscles(self):
        muscles = [
            {"name": "Tríceps", "name_en_us": "Triceps", "level": 0},
            {"name": "Pecho", "name_en_us": "Chest", "level": 0}
        ]
        main_en, main_es, secondary = extract_muscles(muscles)
        self.assertEqual(main_en, "Triceps")
        self.assertEqual(main_es, "Tríceps")
        self.assertEqual(secondary, ["Pecho"])

    def test_ignores_level_1_muscles(self):
        muscles = [
            {"name": "Tríceps", "name_en_us": "Triceps", "level": 0},
            {"name": "Cabeza larga", "level": 1}
        ]
        main_en, main_es, secondary = extract_muscles(muscles)
        self.assertEqual(main_en, "Triceps")
        self.assertEqual(secondary, [])

    def test_empty_list(self):
        main_en, main_es, secondary = extract_muscles([])
        self.assertEqual(main_en, "")
        self.assertEqual(main_es, "")
        self.assertEqual(secondary, [])


class TestClassifyPatternByKeywords(unittest.TestCase):
    """Test classify_pattern_by_keywords function"""

    def test_arm_pattern(self):
        self.assertEqual(classify_pattern_by_keywords("Barbell Curl", "barbell-curl"), "arm")
        self.assertEqual(classify_pattern_by_keywords("Tricep Extension", "tricep-extension"), "arm")

    def test_push_h_pattern(self):
        self.assertEqual(classify_pattern_by_keywords("Bench Press", "bench-press"), "push_h")
        self.assertEqual(classify_pattern_by_keywords("Push-up", "push-up"), "push_h")

    def test_squat_pattern(self):
        self.assertEqual(classify_pattern_by_keywords("Squat", "squat"), "squat")
        self.assertEqual(classify_pattern_by_keywords("Leg Press", "leg-press"), "squat")

    def test_hinge_pattern(self):
        self.assertEqual(classify_pattern_by_keywords("Deadlift", "deadlift"), "hinge")
        self.assertEqual(classify_pattern_by_keywords("Hip Thrust", "hip-thrust"), "hinge")

    def test_core_pattern(self):
        self.assertEqual(classify_pattern_by_keywords("Plank", "plank"), "core")
        self.assertEqual(classify_pattern_by_keywords("Crunch", "crunch"), "core")

    def test_no_match_returns_none(self):
        self.assertIsNone(classify_pattern_by_keywords("Unknown Exercise", "unknown-exercise"))


class TestGetRole(unittest.TestCase):
    """Test get_role function"""

    def test_core_pattern_returns_core(self):
        self.assertEqual(get_role("core", "Plank"), "core")

    def test_compound_patterns(self):
        self.assertEqual(get_role("squat", "Barbell Squat"), "compound")
        self.assertEqual(get_role("hinge", "Deadlift"), "compound")

    def test_isolation_patterns(self):
        self.assertEqual(get_role("arm", "Bicep Curl"), "isolation")
        self.assertEqual(get_role("accessory", "Calf Raise"), "isolation")

    def test_isolation_override(self):
        self.assertEqual(get_role("push_v", "Lateral Raise"), "isolation")
        self.assertEqual(get_role("push_h", "Dumbbell Fly"), "isolation")


class TestFormatSecondaryMusclesForPostgres(unittest.TestCase):
    """Test format_secondary_muscles_for_postgres function"""

    def test_empty_list(self):
        self.assertEqual(format_secondary_muscles_for_postgres([]), "{}")

    def test_single_muscle(self):
        self.assertEqual(format_secondary_muscles_for_postgres(["Pecho"]), '{"Pecho"}')

    def test_multiple_muscles(self):
        result = format_secondary_muscles_for_postgres(["Pecho", "Hombros", "Tríceps"])
        self.assertEqual(result, '{"Pecho","Hombros","Tríceps"}')


class TestDeduplicationStats(unittest.TestCase):
    """Test DeduplicationStats dataclass"""

    def test_initialization(self):
        stats = DeduplicationStats()
        self.assertEqual(stats.total_raw, 0)
        self.assertEqual(stats.unique_slugs, 0)
        self.assertEqual(stats.duplicates_removed, 0)
        self.assertEqual(stats.files_processed, 0)

    def test_manual_values(self):
        stats = DeduplicationStats(
            total_raw=100,
            unique_slugs=80,
            duplicates_removed=20,
            files_processed=5
        )
        self.assertEqual(stats.total_raw, 100)
        self.assertEqual(stats.unique_slugs, 80)
        self.assertEqual(stats.duplicates_removed, 20)
        self.assertEqual(stats.files_processed, 5)


class TestCollectUniqueExercises(unittest.TestCase):
    """Test collect_unique_exercises function"""

    def test_single_file_no_duplicates(self):
        with tempfile.TemporaryDirectory() as tmp_dir:
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
            json_file = Path(tmp_dir) / "test.json"
            json_file.write_text(json.dumps(exercises))

            unique, stats = collect_unique_exercises([json_file])

            self.assertEqual(stats.files_processed, 1)
            self.assertEqual(stats.total_raw, 2)
            self.assertEqual(stats.unique_slugs, 2)
            self.assertEqual(stats.duplicates_removed, 0)
            self.assertEqual(len(unique), 2)
            self.assertIn("barbell-curl", unique)
            self.assertIn("bench-press", unique)

    def test_multiple_files_with_duplicates(self):
        with tempfile.TemporaryDirectory() as tmp_dir:
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
            file1 = Path(tmp_dir) / "biceps.json"
            file1.write_text(json.dumps(exercises1))

            # File 2 (duplicate curl + new exercise)
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
            file2 = Path(tmp_dir) / "chest.json"
            file2.write_text(json.dumps(exercises2))

            unique, stats = collect_unique_exercises([file1, file2])

            self.assertEqual(stats.files_processed, 2)
            self.assertEqual(stats.total_raw, 3)
            self.assertEqual(stats.unique_slugs, 2)
            self.assertEqual(stats.duplicates_removed, 1)
            self.assertEqual(len(unique), 2)


class TestTransformExercise(unittest.TestCase):
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

        self.assertEqual(result['exercise_id'], "ex_barbell_curl")
        self.assertEqual(result['name'], "Barbell Curl")
        self.assertEqual(result['spanish_name'], "Curl con barra")
        self.assertEqual(result['main_muscle'], "Biceps")
        self.assertEqual(result['Músculo Principal'], "Bíceps")
        self.assertEqual(result['equipment'], "barbell")
        self.assertEqual(result['level'], "Intermedio")
        self.assertEqual(result['pattern'], "arm")
        self.assertEqual(result['role'], "isolation")

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

        self.assertEqual(result['main_muscle'], "Chest")
        self.assertEqual(result['secondary_muscles'], '{"Tríceps","Hombros"}')

    def test_unilateral_detection(self):
        exercise = {
            "name": "Curl con una mano",
            "target_url": {"male": "dumbbell/male/biceps/single-arm-curl"},
            "muscles": [{"name": "Bíceps", "level": 0}],
            "category": {"name_en_us": "Dumbbells"},
            "difficulty": {"name": "Intermedio"}
        }

        result = transform_exercise(exercise)

        self.assertEqual(result['unilateral'], "Yes")


class TestEndToEndDeduplication(unittest.TestCase):
    """Integration tests for the full deduplication pipeline"""

    def test_realistic_scenario(self):
        with tempfile.TemporaryDirectory() as tmp_dir:
            barbell_curl = {
                "name": "Curl con barra",
                "target_url": {"male": "barbell/male/biceps/barbell-curl"},
                "muscles": [{"name": "Bíceps", "level": 0}],
                "category": {"name_en_us": "Barbell"},
                "difficulty": {"name": "Intermedio"}
            }

            bench_press = {
                "name": "Press de banca",
                "target_url": {"male": "barbell/male/chest/bench-press"},
                "muscles": [{"name": "Pecho", "level": 0}, {"name": "Tríceps", "level": 0}],
                "category": {"name_en_us": "Barbell"},
                "difficulty": {"name": "Intermedio"}
            }

            # Create 3 files with overlapping exercises
            file1 = Path(tmp_dir) / "biceps.json"
            file1.write_text(json.dumps([barbell_curl]))

            file2 = Path(tmp_dir) / "triceps.json"
            file2.write_text(json.dumps([bench_press]))

            file3 = Path(tmp_dir) / "arms.json"
            file3.write_text(json.dumps([barbell_curl, bench_press]))

            # Collect unique
            unique, stats = collect_unique_exercises([file1, file2, file3])

            self.assertEqual(stats.files_processed, 3)
            self.assertEqual(stats.total_raw, 4)
            self.assertEqual(stats.unique_slugs, 2)
            self.assertEqual(stats.duplicates_removed, 2)

            # Transform unique exercises
            transformed = []
            for slug, (exercise, source_file) in unique.items():
                transformed.append(transform_exercise(exercise))

            self.assertEqual(len(transformed), 2)

            # Check IDs are consistent
            ids = {ex['exercise_id'] for ex in transformed}
            self.assertIn("ex_barbell_curl", ids)
            self.assertIn("ex_bench_press", ids)


if __name__ == "__main__":
    # Run tests with verbose output
    unittest.main(verbosity=2)
