"""Canonical contract for draft_routines.draft_data JSONB.

Single source of truth for the JSON shape exchanged between the Kairos agent
(producer) and workout-tracker-back (consumer). The producer validates against
this schema before persisting to Supabase, which guarantees the Go service
receives well-typed data regardless of LLM output quirks.
"""

from __future__ import annotations

from pydantic import AliasChoices, BaseModel, ConfigDict, Field, field_validator


def _stringify(value: object) -> str:
    if value is None:
        return ""
    return str(value)


class DraftAlternative(BaseModel):
    model_config = ConfigDict(populate_by_name=True, extra="ignore")

    exercise_id: str
    spanish_name: str
    main_muscle: str = ""
    # Accept both legacy `link` (from _fetch_alternatives_for_exercise) and
    # the canonical `video_link` so historical and new data converge on one key.
    video_link: str = Field(
        default="",
        validation_alias=AliasChoices("video_link", "link"),
    )


class DraftExercise(BaseModel):
    model_config = ConfigDict(extra="ignore")

    exercise_id: str
    spanish_name: str
    pattern: str
    role: str
    sets: int
    reps: str
    rir: str
    rest_seconds: int
    exercise_order: int
    main_muscle: str = ""
    video_link: str = ""
    alternatives: list[DraftAlternative] = Field(default_factory=list)

    @field_validator("rir", "reps", mode="before")
    @classmethod
    def _coerce_str(cls, value: object) -> str:
        return _stringify(value)


class DraftDay(BaseModel):
    model_config = ConfigDict(extra="ignore")

    day_number: int
    title: str
    exercises: list[DraftExercise]


class DraftData(BaseModel):
    model_config = ConfigDict(extra="ignore")

    week_schedule: str
    goal: str
    level: str
    days: list[DraftDay]


def normalize_draft_data(raw: dict) -> dict:
    """Validate raw draft dict and return a normalized dict ready for JSONB insert."""
    return DraftData.model_validate(raw).model_dump(by_alias=True)
