"""Mock day templates mirroring GymBot's template_days + day_requirements tables."""

from src.shared.types import DayTemplate

MOCK_TEMPLATES: dict[str, DayTemplate] = {
    "Full Body A": {
        "template_name": "Full Body A",
        "patterns": ["push", "squat", "pull", "core"],
    },
    "Full Body B": {
        "template_name": "Full Body B",
        "patterns": ["hip_hinge", "push", "pull", "isolation"],
    },
}


def get_template(name: str) -> DayTemplate | None:
    """Fetch a day template by name."""
    return MOCK_TEMPLATES.get(name)


def get_all_templates() -> list[DayTemplate]:
    """Return all available templates."""
    return list(MOCK_TEMPLATES.values())
