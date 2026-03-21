# Quickstart: Onboarding KYC (Case 5)

**Feature**: 001-onboarding-kyc | **Date**: 2026-03-15

## Prerequisites

- Python 3.11+
- `langgraph-skeleton/.venv` virtual environment activated
- `.env` file with `GOOGLE_API_KEY` (for Gemini)
- `.env` file with `SUPABASE_URL` and `SUPABASE_ANON_KEY` (for live mode)

## Setup

```bash
cd langgraph-skeleton
source .venv/bin/activate
pip install -e ".[dev]"
```

## Running

### Option 1: FastAPI Server (recommended for testing)

```bash
uvicorn server:app --reload --port 8000
```

Open Swagger UI at `http://localhost:8000/docs` and use the Case 5 endpoints.

**Example multi-turn flow via curl**:

```bash
# Turn 1: Initial greeting
curl -X POST http://localhost:8000/case5/kyc/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Hola, quiero empezar", "phone_number": "573001234567", "display_name": "Camilo"}'

# Turn 2: Answer goal + experience
curl -X POST http://localhost:8000/case5/kyc/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Quiero ganar masa muscular, tengo 3 años de experiencia", "phone_number": "573001234567"}'

# Turn 3: Training environment
curl -X POST http://localhost:8000/case5/kyc/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Entreno en gym, 3 días a la semana, por las mañanas", "phone_number": "573001234567"}'

# Turn 4: Body metrics
curl -X POST http://localhost:8000/case5/kyc/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Soy hombre, 27 años, 171 cm, 67 kg", "phone_number": "573001234567"}'

# Turn 5: Health status
curl -X POST http://localhost:8000/case5/kyc/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "No tengo ninguna lesión ni condición", "phone_number": "573001234567"}'

# Check history
curl http://localhost:8000/case5/kyc/history?thread_id=kyc_573001234567
```

### Option 2: Standalone Runner

```bash
python -m cases.case5_onboarding_kyc.run
```

Interactive CLI that simulates the multi-turn KYC conversation.

### Option 3: Tests

```bash
# Run Case 5 tests only
pytest tests/test_case5.py -v

# Run all tests
pytest tests/ -v
```

## File Structure

```
cases/case5_onboarding_kyc/
├── __init__.py
├── state.py              # KYCState TypedDict
├── nodes.py              # Node functions (check_user, kyc_agent, etc.)
├── prompts.py            # Spanish system prompts for each turn
├── tools_supabase.py     # Supabase tools (lookup_user, save_profile)
├── graph.py              # KYC graph with InMemorySaver (mock)
├── graph_live.py         # KYC graph with Supabase (live)
└── run.py                # Standalone runner
```

## Key Behaviors to Test

1. **New user detection**: Send from an unregistered phone → Kairos starts KYC
2. **Existing user redirect**: Send from a registered phone → Kairos says "ya tienes perfil"
3. **Multi-value messages**: "3 días, soy intermedio" → both fields registered
4. **Progress indicators**: Each response includes "Pregunta X de 5"
5. **Session resumption**: Send messages with same `thread_id` → conversation continues
6. **Profile confirmation**: After Turn 5, Kairos shows summary → user says "sí"
7. **Correction flow**: User says "no, mi objetivo está mal" → targeted correction
8. **Health classification**: "Tengo dolor de rodilla" → health code B
9. **Severe health (code E)**: "Tengo problemas cardíacos" → route to trainer

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `GOOGLE_API_KEY` | Yes | Gemini API key |
| `SUPABASE_URL` | Live only | Supabase project URL |
| `SUPABASE_ANON_KEY` | Live only | Supabase anonymous key |
