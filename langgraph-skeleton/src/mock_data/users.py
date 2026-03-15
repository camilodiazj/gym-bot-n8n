"""Mock users mirroring GymBot's users + users_gym_profile tables."""

from src.shared.types import UserProfile

MOCK_USERS: dict[str, UserProfile] = {
    "camilo-001": {
        "user_id": "camilo-001",
        "full_name": "Camilo Diaz",
        "primary_goal": "Ganar masa muscular",
        "health_status": "A",  # Sin restricciones
        "level": "Intermedio",
        "days_available": 3,
    },
    "ana-002": {
        "user_id": "ana-002",
        "full_name": "Ana Martinez",
        "primary_goal": "Bajar grasa",
        "health_status": "C",  # Upper body issues — evitar overhead pressing
        "level": "Principiante",
        "days_available": 2,
    },
    "carlos-003": {
        "user_id": "carlos-003",
        "full_name": "Carlos Rodriguez",
        "primary_goal": "Mejorar fuerza",
        "health_status": "B",  # Lower body issues — evitar high impact
        "level": "Avanzado",
        "days_available": 4,
    },
}


def get_user(user_id: str) -> UserProfile | None:
    """Fetch a mock user by ID. Returns None if not found."""
    return MOCK_USERS.get(user_id)
