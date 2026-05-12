"""Tests for the canonical draft_routines contract.

Verifies that normalization absorbs LLM quirks (int rir, legacy `link` key,
extra fields) and that genuinely-missing required fields raise ValidationError.
"""

import pytest
from pydantic import ValidationError

from cases.case6_unified_agent.draft_schema import (
    DraftAlternative,
    DraftData,
    DraftExercise,
    normalize_draft_data,
)


def _valid_exercise(**overrides) -> dict:
    base = {
        "exercise_id": "ex_032",
        "spanish_name": "Sentadilla búlgara",
        "pattern": "squat",
        "role": "compound",
        "sets": 3,
        "reps": "8-10",
        "rir": "1-2",
        "rest_seconds": 120,
        "exercise_order": 1,
        "alternatives": [],
    }
    base.update(overrides)
    return base


def _valid_draft(exercise_overrides=None) -> dict:
    ex = _valid_exercise(**(exercise_overrides or {}))
    return {
        "week_schedule": "fb_2",
        "goal": "Ganar masa muscular",
        "level": "Intermedio",
        "days": [
            {"day_number": 1, "title": "Full Body A", "exercises": [ex]},
        ],
    }


# ─────────────────────── rir coercion ───────────────────────

def test_rir_int_is_coerced_to_string():
    raw = _valid_draft({"rir": 2})
    out = normalize_draft_data(raw)
    assert out["days"][0]["exercises"][0]["rir"] == "2"


def test_rir_float_is_coerced_to_string():
    raw = _valid_draft({"rir": 1.5})
    out = normalize_draft_data(raw)
    assert out["days"][0]["exercises"][0]["rir"] == "1.5"


def test_rir_string_passes_through():
    raw = _valid_draft({"rir": "1-2"})
    out = normalize_draft_data(raw)
    assert out["days"][0]["exercises"][0]["rir"] == "1-2"


def test_reps_int_is_coerced_to_string():
    raw = _valid_draft({"reps": 10})
    out = normalize_draft_data(raw)
    assert out["days"][0]["exercises"][0]["reps"] == "10"


# ─────────────────── alternative link → video_link ───────────────────

def test_alternative_legacy_link_becomes_video_link():
    alt = {
        "exercise_id": "ex_002",
        "spanish_name": "Sentadilla Hack",
        "main_muscle": "Quads",
        "link": "https://musclewiki.com/es-es/exercise/machine-hack-squat",
    }
    raw = _valid_draft({"alternatives": [alt]})
    out = normalize_draft_data(raw)
    serialized_alt = out["days"][0]["exercises"][0]["alternatives"][0]
    assert serialized_alt["video_link"] == alt["link"]
    assert "link" not in serialized_alt


def test_alternative_video_link_passes_through():
    alt = {
        "exercise_id": "ex_002",
        "spanish_name": "Sentadilla Hack",
        "video_link": "https://example.com/video",
    }
    out_alt = DraftAlternative.model_validate(alt).model_dump(by_alias=True)
    assert out_alt["video_link"] == "https://example.com/video"


# ─────────────────── extras are dropped, missing fields raise ───────────────────

def test_unknown_fields_are_silently_dropped():
    raw = _valid_draft({"weird_extra_field": "ignored"})
    out = normalize_draft_data(raw)
    assert "weird_extra_field" not in out["days"][0]["exercises"][0]


def test_missing_required_field_raises_validation_error():
    raw = _valid_draft()
    del raw["days"][0]["exercises"][0]["pattern"]
    with pytest.raises(ValidationError):
        normalize_draft_data(raw)


def test_missing_top_level_field_raises():
    raw = _valid_draft()
    del raw["goal"]
    with pytest.raises(ValidationError):
        normalize_draft_data(raw)


# ─────────────────── round-trip preserves required structure ───────────────────

def test_full_draft_round_trip():
    raw = {
        "week_schedule": "fb_3",
        "goal": "Mejorar fuerza",
        "level": "Avanzado",
        "days": [
            {
                "day_number": 1,
                "title": "Full Body A",
                "exercises": [
                    {
                        "exercise_id": "ex_001",
                        "spanish_name": "Sentadilla",
                        "pattern": "squat",
                        "role": "compound",
                        "sets": 4,
                        "reps": "6-8",
                        "rir": 2,
                        "rest_seconds": 180,
                        "exercise_order": 1,
                        "alternatives": [
                            {
                                "exercise_id": "ex_002",
                                "spanish_name": "Sentadilla Hack",
                                "main_muscle": "Quads",
                                "link": "https://x.example/hack",
                            }
                        ],
                    }
                ],
            }
        ],
    }
    out = normalize_draft_data(raw)
    ex = out["days"][0]["exercises"][0]
    assert ex["rir"] == "2"
    assert ex["alternatives"][0]["video_link"] == "https://x.example/hack"
    assert ex["sets"] == 4
    assert ex["rest_seconds"] == 180


def test_none_rir_normalises_to_empty_string():
    raw = _valid_draft({"rir": None})
    out = normalize_draft_data(raw)
    assert out["days"][0]["exercises"][0]["rir"] == ""
