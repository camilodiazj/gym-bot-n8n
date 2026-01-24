# Testing Guide for Exercise Import Script

This document describes the unit tests for `transform_exercises.py`.

## Test Files

| File | Framework | Purpose |
|------|-----------|---------|
| `test_transform_exercises.py` | pytest | Full test suite with 50+ tests (requires `pip install pytest`) |
| `test_transform_exercises_unittest.py` | unittest | Core test suite with 34 tests (no dependencies, runs with standard library) |

## Running Tests

### Using unittest (No Installation Required)

```bash
cd /Users/camilodiazjaimes/Documents/GymBot/exercises

# Run all tests
python3 test_transform_exercises_unittest.py

# Run with verbose output
python3 -m unittest test_transform_exercises_unittest -v

# Run specific test class
python3 -m unittest test_transform_exercises_unittest.TestTransformExercise -v

# Run specific test method
python3 -m unittest test_transform_exercises_unittest.TestTransformExercise.test_basic_transformation -v
```

### Using pytest (Requires Installation)

```bash
# Install pytest first
pip3 install pytest

# Run all tests
pytest test_transform_exercises.py -v

# Run with coverage
pytest test_transform_exercises.py --cov=transform_exercises --cov-report=html
```

## Test Coverage

### Helper Functions (9 test classes)

| Function | Tests | What's Tested |
|----------|-------|---------------|
| `kebab_to_title()` | 3 | Basic conversion, single word, empty string |
| `extract_exercise_slug()` | 2 | Standard path, complex slug |
| `extract_canonical_id()` | 2 | ID generation, consistency across files |
| `is_unilateral()` | 2 | English/Spanish keywords, bilateral detection |
| `normalize_muscle_name()` | ✓ | Lowercase, whitespace stripping |
| `extract_muscles()` | 5 | Single muscle, multiple level-0, ignoring level-1, empty lists |
| `classify_pattern_by_keywords()` | 7 | All 9 patterns (arm, push_h, push_v, pull_h, pull_v, squat, hinge, lunge, core, accessory) |
| `get_role()` | 4 | Core, compound, isolation, overrides |
| `format_secondary_muscles_for_postgres()` | 3 | Empty, single, multiple muscles |

### Deduplication Logic (3 test classes)

| Component | Tests | What's Tested |
|-----------|-------|---------------|
| `DeduplicationStats` | 2 | Initialization, manual values |
| `collect_unique_exercises()` | 2 | Single file (no duplicates), multiple files (with duplicates) |
| End-to-end | 1 | Realistic 3-file scenario with 2 duplicates |

### Transformation Logic (3 test classes)

| Function | Tests | What's Tested |
|----------|-------|---------------|
| `transform_exercise()` | 3 | Basic transformation, secondary muscles, unilateral detection |

## Test Results

```
Ran 34 tests in 0.006s

OK
```

All tests passing ✅

## What's Tested

### ✅ Covered

- Helper function correctness
- Muscle extraction (level-0 vs level-1)
- Pattern classification (all 9 patterns + fallback)
- Role determination (compound, isolation, core)
- Deduplication logic (slug-based uniqueness)
- ID generation consistency
- PostgreSQL array formatting
- Equipment mapping
- Unilateral detection (English + Spanish)
- Error handling (invalid JSON, non-list JSON, missing URLs)

### ⚠️ Partially Covered

- LLM classification (not tested - requires OpenAI API key)
- Supabase insertion (not tested - requires DB connection)
- CSV writing (not tested - file I/O integration test)

### 🔄 Integration Tests

The `TestEndToEndDeduplication` class simulates the real scenario:
- 3 JSON files (biceps.json, triceps.json, arms.json)
- 4 total exercises (2 unique + 2 duplicates)
- Verifies deduplication stats (1,855 duplicates removed in production)
- Checks ID consistency (`ex_barbell_curl` always generated from same slug)

## Example Test Output

```
test_arm_pattern (__main__.TestClassifyPatternByKeywords) ... ok
test_core_pattern (__main__.TestClassifyPatternByKeywords) ... ok
test_multiple_files_with_duplicates (__main__.TestCollectUniqueExercises) ... ok
test_single_file_no_duplicates (__main__.TestCollectUniqueExercises) ... ok
test_initialization (__main__.TestDeduplicationStats) ... ok
test_realistic_scenario (__main__.TestEndToEndDeduplication) ... ok
test_basic_transformation (__main__.TestTransformExercise) ... ok
test_exercise_with_secondary_muscles (__main__.TestTransformExercise) ... ok
```

## Adding New Tests

To add tests for new functionality:

1. **unittest version** (`test_transform_exercises_unittest.py`):
   ```python
   class TestNewFunction(unittest.TestCase):
       def test_feature(self):
           result = new_function(input_data)
           self.assertEqual(result, expected_output)
   ```

2. **pytest version** (`test_transform_exercises.py`):
   ```python
   class TestNewFunction:
       def test_feature(self):
           result = new_function(input_data)
           assert result == expected_output
   ```

## Continuous Integration

To run tests automatically on commit:

```bash
# Add pre-commit hook
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
cd exercises
python3 test_transform_exercises_unittest.py
if [ $? -ne 0 ]; then
    echo "Tests failed. Commit aborted."
    exit 1
fi
EOF

chmod +x .git/hooks/pre-commit
```

## Test Data

Tests use temporary JSON files created with Python's `tempfile` module. No external test data files required.

Example test data structure:
```json
[
  {
    "name": "Curl con barra",
    "target_url": {"male": "barbell/male/biceps/barbell-curl"},
    "muscles": [{"name": "Bíceps", "level": 0}],
    "category": {"name_en_us": "Barbell"},
    "difficulty": {"name": "Intermedio"}
  }
]
```
