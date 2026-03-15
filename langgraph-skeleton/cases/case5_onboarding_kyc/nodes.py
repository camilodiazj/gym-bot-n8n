"""Node functions for the Onboarding KYC StateGraph.

Each node is an async function: (KYCState) -> dict
The returned dict is merged into KYCState by the graph runtime.
"""

import json
import re
from datetime import datetime, timezone

from langchain_core.messages import AIMessage, HumanMessage, SystemMessage

from src.shared.llm import get_llm
from cases.case5_onboarding_kyc.state import (
    KYCState,
    KYC_FIELDS,
    ALL_KYC_FIELDS,
    compute_current_turn,
    is_kyc_complete,
)
from cases.case5_onboarding_kyc.prompts import (
    KYC_MASTER_PROMPT,
    TURN_PROMPTS,
    HEALTH_CLASSIFIER_PROMPT,
    ROUTE_TO_TRAINER_PROMPT,
    RESUMPTION_GREETING,
    CORRECTION_PROMPT,
    format_collected_summary,
)


# ═══════════════ CHECK_USER (T004) ═══════════════

async def check_user(state: KYCState) -> dict:
    """Query for existing user by phone number.

    Mock version: always returns is_new_user=True.
    Live version (graph_live.py) queries Supabase.
    """
    phone = state.get("phone_number", "")
    display_name = state.get("display_name", "")

    # Mock: always treat as new user
    return {
        "is_new_user": True,
        "display_name": display_name if display_name else "",
        "last_interaction_at": datetime.now(timezone.utc).isoformat(),
    }


# ═══════════════ CHECK_STATUS (T005) ═══════════════

async def check_status(state: KYCState) -> dict:
    """Determine KYC progress and route the graph.

    Inspects collected_data to compute current_turn (1-5).
    Sets is_complete=True when all required fields are present.
    Updates last_interaction_at timestamp.
    Detects resumption if gap > 30 min since last interaction.
    Detects correction intent from user's latest message (US3).
    """
    collected = state.get("collected_data", {})
    current_turn = compute_current_turn(collected)
    complete = is_kyc_complete(collected)

    # Detect resumption: gap > 30 min since last interaction
    is_resumption = False
    last_at = state.get("last_interaction_at", "")
    if last_at and collected:
        try:
            last_dt = datetime.fromisoformat(last_at)
            gap = (datetime.now(timezone.utc) - last_dt).total_seconds()
            is_resumption = gap > 1800  # 30 minutes
        except (ValueError, TypeError):
            pass

    # Detect confirmation or correction from latest USER message (not AI)
    needs_correction = False
    correction_field = ""
    profile_confirmed = False

    if collected and state.get("awaiting_confirmation"):
        messages = state.get("messages", [])
        # Find the last HumanMessage — not messages[-1] which is the AIMessage from kyc_agent
        last_human_msg = ""
        for msg in reversed(messages):
            if isinstance(msg, HumanMessage):
                last_human_msg = msg.content.lower()
                break

        if last_human_msg:
            correction_keywords = ["cambiar", "corregir", "mal", "error", "incorrecto", "equivocado"]
            confirm_keywords = ["sí", "si", "correcto", "bien", "perfecto", "listo", "todo bien", "ok", "dale", "confirmo"]
            if any(kw in last_human_msg for kw in correction_keywords):
                needs_correction = True
                # Try to detect which field from known labels
                field_hints = {
                    "objetivo": "primary_goal",
                    "experiencia": "training_experience",
                    "días": "days_available",
                    "dia": "days_available",
                    "horario": "preferred_schedule",
                    "lugar": "training_environment",
                    "casa": "training_environment",
                    "gym": "training_environment",
                    "equipo": "home_equipment",
                    "sexo": "biological_sex",
                    "edad": "age",
                    "estatura": "height_cm",
                    "altura": "height_cm",
                    "peso": "weight_kg",
                    "salud": "health_status",
                }
                for hint, field in field_hints.items():
                    if hint in last_human_msg:
                        correction_field = field
                        break

                # Direct extraction for common corrections to avoid
                # relying on LLM re-extraction (which is unreliable)
                updated_collected = dict(collected)
                if correction_field == "training_environment" or "casa" in last_human_msg or "home" in last_human_msg:
                    if any(kw in last_human_msg for kw in ["casa", "home", "en casa"]):
                        updated_collected["training_environment"] = "HOME"
                        # Extract equipment from the same message
                        equip_keywords = {
                            "mancuerna": "mancuernas", "banda": "bandas elásticas",
                            "barra": "barra", "disco": "discos", "rack": "rack",
                            "banca": "banca", "peso corporal": "peso corporal",
                        }
                        found_equip = []
                        for kw, label in equip_keywords.items():
                            if kw in last_human_msg:
                                found_equip.append(label)
                        if found_equip:
                            updated_collected["home_equipment"] = ", ".join(found_equip)
                        elif "solo mi cuerpo" in last_human_msg or "peso corporal" in last_human_msg:
                            updated_collected["home_equipment"] = "peso corporal"
                        needs_correction = False  # Handled inline, skip kyc_agent
                        collected = updated_collected
                    elif any(kw in last_human_msg for kw in ["gym", "gimnasio"]):
                        updated_collected["training_environment"] = "GYM"
                        updated_collected.pop("home_equipment", None)
                        needs_correction = False
                        collected = updated_collected
                elif correction_field == "weight_kg":
                    # Try to extract weight number
                    import re as _re
                    weight_match = _re.search(r'(\d+)\s*(?:kg|kilo)', last_human_msg)
                    if weight_match:
                        updated_collected["weight_kg"] = int(weight_match.group(1))
                        needs_correction = False
                        collected = updated_collected
            elif any(kw in last_human_msg for kw in confirm_keywords):
                profile_confirmed = True

    # Recompute completion after potential inline corrections
    current_turn = compute_current_turn(collected)
    complete = is_kyc_complete(collected)

    result = {
        "current_turn": current_turn if not complete else 5,
        "is_complete": complete,
        "collected_data": collected,
        "last_interaction_at": datetime.now(timezone.utc).isoformat(),
        "needs_correction": needs_correction,
        "correction_field": correction_field,
    }

    if profile_confirmed:
        result["profile_confirmed"] = True
        result["awaiting_confirmation"] = False

    # If correction was handled inline, route back to confirm_profile
    if not needs_correction and state.get("awaiting_confirmation") and not profile_confirmed and collected != state.get("collected_data", {}):
        result["is_complete"] = True
        result["awaiting_confirmation"] = False
        # Will route to confirm_profile via route_after_check_status (is_complete=True)

    if needs_correction:
        result["awaiting_confirmation"] = False

    if is_resumption:
        result["is_resumption"] = True

    return result


# ═══════════════ KYC_AGENT (T006) ═══════════════

async def kyc_agent(state: KYCState) -> dict:
    """Gemini conducts conversational KYC, one turn at a time.

    Uses turn-specific prompts and extracts field values from
    the LLM response via structured output.
    """
    llm = get_llm(temperature=0.7)
    collected = state.get("collected_data", {})
    current_turn = compute_current_turn(collected)
    display_name = state.get("display_name", "")

    # Build the system prompt with turn-specific instructions
    turn_prompt = TURN_PROMPTS.get(current_turn, TURN_PROMPTS[1])
    collected_summary = format_collected_summary(collected)

    # Check for correction mode (US3)
    if state.get("needs_correction", False):
        correction_field = state.get("correction_field", "")
        turn_prompt = CORRECTION_PROMPT.format(
            collected_summary=collected_summary,
            correction_field=correction_field or "no especificado",
        )

    # Include next turn's question so Kairos asks it after extracting current data
    next_turn = current_turn + 1
    next_turn_prompt = TURN_PROMPTS.get(next_turn)
    if next_turn_prompt and not state.get("needs_correction", False):
        turn_prompt += (
            "\n\nIMPORTANTE: Después de reconocer los datos del usuario, "
            "DEBES hacer la pregunta del siguiente turno:\n" + next_turn_prompt
        )

    system_content = KYC_MASTER_PROMPT.format(
        current_turn=current_turn,
        display_name=display_name or "amigo",
        collected_summary=collected_summary,
        turn_prompt=turn_prompt,
    )

    # Add resumption context (US2) if returning after inactivity
    if state.get("is_resumption", False):
        last_turn = max(1, current_turn - 1)
        system_content += "\n\n" + RESUMPTION_GREETING.format(
            last_turn=last_turn,
            display_name=display_name or "amigo",
            next_turn=current_turn,
        )

    # Tell the LLM which fields are already collected so it skips redundant questions
    already_collected = list(collected.keys())
    skip_note = ""
    if already_collected:
        skip_note = (
            f"\n\nCAMPOS YA RECOLECTADOS (NO preguntes por estos de nuevo): "
            f"{', '.join(already_collected)}\n"
            "Si el turno actual pide un campo que ya tienes, salta directamente "
            "a la pregunta del SIGUIENTE turno que tenga campos pendientes."
        )

    # Add extraction instructions — allow ALL fields, not just current turn
    current_turn_fields = KYC_FIELDS.get(current_turn, [])
    all_field_names = ", ".join(sorted(ALL_KYC_FIELDS))
    system_content += (
        f"{skip_note}"
        "\n\nDESPUÉS de tu respuesta conversacional, incluye un bloque JSON "
        "al final con los datos extraídos del mensaje del usuario.\n"
        "Formato: EXTRACTED_DATA: {\"campo\": \"valor\", ...}\n"
        "Si el usuario no proporcionó datos nuevos, usa: EXTRACTED_DATA: {}\n"
        f"Campos PRIORITARIOS para este turno: {', '.join(current_turn_fields)}\n"
        f"PERO si el usuario da datos de CUALQUIER campo, extráelos TODOS: {all_field_names}\n"
        "\nREGLAS DE EXTRACCIÓN IMPORTANTES:\n"
        "- Si el usuario dice 'gym', 'gimnasio', 'al gym' → training_environment: 'GYM'\n"
        "- Si dice 'casa', 'home', 'en casa' → training_environment: 'HOME'\n"
        "- Convierte números escritos en texto: 'veintisiete' → 27, 'uno setenta y uno' → 171\n"
        "- Convierte medidas: '1.60 m' → 160, 'un metro sesenta y cinco' → 165\n"
        "- 'F' o 'mujer' o 'femenino' → biological_sex: 'F'\n"
        "- 'M' o 'hombre' o 'masculino' → biological_sex: 'M'\n"
        "- Extrae TODOS los datos mencionados, incluso si son de turnos diferentes.\n"
        "- Para salud: 'sano', 'completamente sano', 'sin nada', 'estoy bien', "
        "'no tengo nada', 'sin problemas' → health_status: 'Sin restricciones'\n"
        "- CUALQUIER mención de salud/lesiones/condiciones → health_status: texto exacto del usuario\n"
        "- Para training_experience: '6 meses' o '6 meses de experiencia' → '6 a 12 meses' (NO 'Menos de 6 meses')\n"
        "- 'un año' o '1 año' → '1 a 3 años' (NO '6 a 12 meses'); '2 años' o '3 años' → '1 a 3 años'; '4+ años' → 'Más de 3 años'\n"
    )

    messages = [SystemMessage(content=system_content)] + state["messages"]
    response = await llm.ainvoke(messages)

    # Parse extracted data from response
    new_data = _parse_extracted_data(response.content)
    updated_collected = {**collected, **new_data}

    # Strip the JSON block from the visible response
    clean_response = _strip_extracted_data(response.content)

    return {
        "messages": [AIMessage(content=clean_response)],
        "collected_data": updated_collected,
        "response": clean_response,
    }


# ═══════════════ CONFIRM_PROFILE (T007) ═══════════════

async def confirm_profile(state: KYCState) -> dict:
    """Present profile summary for user confirmation.

    Uses a deterministic template — no LLM call — to guarantee
    the summary is always shown (eliminates empty-response bug).
    """
    collected = state.get("collected_data", {})

    equipment_line = ""
    if collected.get("training_environment") == "HOME":
        equipment_line = f"\n🏠 Equipo: {collected.get('home_equipment', 'N/A')}"

    health_display = collected.get("health_status", "N/A")
    # Normalize "no issues" health text for display
    hl = health_display.lower().strip()
    no_issues_display = [
        "sin restricciones", "nada", "no", "ninguna", "sano", "sana",
        "perfecto", "perfecta", "estoy bien", "todo bien", "sin problemas",
        "sin lesiones", "no tengo nada", "no tengo ninguna",
    ]
    if hl in no_issues_display or re.search(
        r"(ya se me pas[oó]|ya no me (duele|molesta)|ya est[oá] bien|ahora estoy bien)",
        hl,
    ):
        health_display = "Sin restricciones"

    summary = (
        f"\n✅ ¡Listo! Este es tu perfil:\n\n"
        f"🎯 Objetivo: {collected.get('primary_goal', 'N/A')}\n"
        f"💪 Experiencia: {collected.get('training_experience', 'N/A')}\n"
        f"📅 Días/semana: {collected.get('days_available', 'N/A')}\n"
        f"🕐 Horario: {collected.get('preferred_schedule', 'N/A')}\n"
        f"🏋️ Lugar: {collected.get('training_environment', 'N/A')}"
        f"{equipment_line}\n"
        f"👤 Sexo: {collected.get('biological_sex', 'N/A')} | "
        f"Edad: {collected.get('age', 'N/A')} | "
        f"Estatura: {collected.get('height_cm', 'N/A')}cm | "
        f"Peso: {collected.get('weight_kg', 'N/A')}kg\n"
        f"🏥 Salud: {health_display}\n\n"
        f"¿Todo está correcto? Si algo está mal, dime qué dato quieres corregir."
    )

    return {
        "messages": [AIMessage(content=summary)],
        "response": summary,
        "awaiting_confirmation": True,
    }


# ═══════════════ SAVE_PROFILE (T008) ═══════════════

async def save_profile(state: KYCState) -> dict:
    """Persist completed profile.

    Mock version: just sets profile_confirmed=True.
    Live version (graph_live.py) writes to Supabase.
    """
    collected = state.get("collected_data", {})
    display_name = state.get("display_name", "")
    phone = state.get("phone_number", "")

    # Mock: log the profile that would be saved
    print(f"[MOCK] Saving profile for {display_name} ({phone}): {collected}")

    name_part = f", {display_name}" if display_name else ""
    success_msg = (
        f"✅ ¡Perfil guardado exitosamente{name_part}! "
        "Ahora voy a crear tu rutina personalizada. 💪"
    )

    return {
        "profile_confirmed": True,
        "awaiting_confirmation": False,
        "messages": [AIMessage(content=success_msg)],
        "response": success_msg,
    }


# ═══════════════ HEALTH_CLASSIFIER (T021) ═══════════════

async def health_classifier(state: KYCState) -> dict:
    """Classify health condition into code A-E via Gemini."""
    collected = state.get("collected_data", {})
    health_text = collected.get("health_status", "Sin restricciones")

    # Short-circuit ONLY for unambiguous "no issues" phrases.
    # Uses exact phrase matching to avoid false positives like
    # "diagnosticada" containing "no" or "ano" containing "no".
    no_issues_phrases = [
        r"^sin\b.*\b(lesion|restriccion|problema|nada)",
        r"^no\s+(tengo|hay)\b",
        r"^(nada|ninguna|sano|sana|perfecto|perfecta)$",
        r"^estoy\s+bien$",
        r"^todo\s+bien$",
        r"^sin\s+restricciones$",
    ]
    health_lower = health_text.lower().strip()
    if any(re.search(pat, health_lower) for pat in no_issues_phrases):
        return {
            "health_code": "A",
            "affected_zones": [],
        }

    llm = get_llm(temperature=0.2)
    prompt = HEALTH_CLASSIFIER_PROMPT.format(health_text=health_text)
    response = await llm.ainvoke(prompt)

    # Parse JSON from response
    try:
        result = json.loads(response.content.strip())
        code = result.get("code", "A")
        zones = result.get("zones", [])
    except (json.JSONDecodeError, AttributeError):
        # Fallback: try to extract from text
        code = "A"
        zones = []
        content = response.content.upper()
        for c in ["E", "D", "C", "B"]:
            if f'"CODE": "{c}"' in content or f'"code": "{c.lower()}"' in response.content:
                code = c
                break

    return {
        "health_code": code,
        "affected_zones": zones,
    }


# ═══════════════ ROUTE_TO_TRAINER (T022) ═══════════════

async def route_to_trainer(state: KYCState) -> dict:
    """Health code E — recommend human trainer, do NOT generate routine."""
    affected = state.get("affected_zones", [])
    display_name = state.get("display_name", "")

    llm = get_llm(temperature=0.7)
    prompt = ROUTE_TO_TRAINER_PROMPT.format(
        affected_zones=", ".join(affected) if affected else "condición severa",
    )
    response = await llm.ainvoke(prompt)

    return {
        "route_to_trainer": True,
        "messages": [response],
        "response": response.content,
    }


# ═══════════════ ROUTING FUNCTIONS ═══════════════

def route_after_check_user(state: KYCState) -> str:
    """Route after check_user: new users → kyc_agent, existing → END."""
    if state.get("is_new_user", True):
        return "kyc_agent"
    return "__end__"


def route_after_check_status(state: KYCState) -> str:
    """Route after check_status: continue, complete, confirmed, or correction."""
    if state.get("profile_confirmed", False):
        return "health_classifier"  # User confirmed → skip confirm_profile
    if state.get("needs_correction", False):
        return "kyc_agent"
    if state.get("is_complete", False):
        return "confirm_profile"
    return "__end__"  # Wait for next user message


def route_after_confirm(state: KYCState) -> str:
    """Route after confirm_profile: accepted → health_classifier, rejected → kyc_agent."""
    if state.get("profile_confirmed", False):
        return "health_classifier"
    # Still awaiting or rejected — go back to kyc_agent for correction
    if state.get("needs_correction", False):
        return "kyc_agent"
    return "__end__"  # Awaiting confirmation (wait for next message)


def route_after_health(state: KYCState) -> str:
    """Route after health_classifier: safe (A-D) → save_profile, severe (E) → route_to_trainer."""
    if state.get("health_code", "A") == "E":
        return "route_to_trainer"
    return "save_profile"


# ═══════════════ HELPERS ═══════════════

def _parse_extracted_data(content: str) -> dict:
    """Extract structured data from LLM response.

    Looks for EXTRACTED_DATA: {...} pattern in the response.
    """
    marker = "EXTRACTED_DATA:"
    idx = content.find(marker)
    if idx == -1:
        return {}

    json_str = content[idx + len(marker):].strip()
    # Find the JSON object
    brace_start = json_str.find("{")
    if brace_start == -1:
        return {}

    # Find matching closing brace
    depth = 0
    for i, ch in enumerate(json_str[brace_start:], start=brace_start):
        if ch == "{":
            depth += 1
        elif ch == "}":
            depth -= 1
            if depth == 0:
                try:
                    return json.loads(json_str[brace_start:i + 1])
                except json.JSONDecodeError:
                    return {}
    return {}


def _strip_extracted_data(content: str) -> str:
    """Remove the EXTRACTED_DATA: {...} block from visible response."""
    marker = "EXTRACTED_DATA:"
    idx = content.find(marker)
    if idx == -1:
        return content.strip()
    return content[:idx].strip()
