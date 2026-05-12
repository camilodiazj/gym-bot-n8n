"""End-to-end contract test for the draft swap → approve → workouts flow.

This is the first E2E test in the repo. It exercises the full Go ↔ Kairos ↔
Supabase boundary, which is exactly where BUG-1 lived (workouts created
during `approve` did not reflect the user's swaps). Unit tests in each
service kept passing while the bug shipped, because no individual layer
saw both sides of the contract.

Why this test runs separately from the fast suite
-------------------------------------------------

* It writes to a real Supabase project and a real Kairos Cloud Run service.
* It depends on an LLM round-trip during onboarding, so latency is high
  and the run isn't reproducible byte-for-byte.

To keep the default `pytest` run fast and deterministic, this file is
gated behind two layers:

1. ``pytest.mark.e2e`` so `pytest -m "not e2e"` (CI default) skips it.
2. ``pytest.mark.skip`` so even an explicit ``-m e2e`` run no-ops it
   until the squad opts in. **Remove the skip** once Dev D / the squad
   stand up the staging credentials and a nightly job to run it.

Environment variables expected when re-enabled:
    KAIROS_URL          e.g. https://kairos-agent-…us-central1.run.app
    WORKOUT_API_URL     e.g. https://workout-api-…us-central1.run.app
    SUPABASE_DB_URL     postgres connection string (used for assertion +
                        teardown; the test mutates a reserved phone fixture)

Reserved fixture phone:
    570000000099  /  Test_E2E_DevD
"""

from __future__ import annotations

import os
import re

import pytest

pytestmark = [pytest.mark.e2e]

TEST_PHONE = "570000000099"
TEST_NAME = "Test_E2E_DevD"


def _parse_db_url(raw: str) -> dict:
    """Split a postgres URL whose password may contain ``@`` characters.

    The Supabase pooler URL embeds a password that itself contains ``@``,
    which urlparse handles incorrectly. We carve out the components by
    regex to feed psycopg's keyword-arg connect API.
    """
    m = re.match(r"^postgresql://([^:]+):(.+)@([^@/]+):(\d+)/(.+)$", raw)
    if not m:
        raise ValueError("unrecognised SUPABASE_DB_URL shape")
    user, password, host, port, dbname = m.groups()
    return {
        "user": user,
        "password": password,
        "host": host,
        "port": int(port),
        "dbname": dbname,
        "sslmode": "require",
    }


@pytest.fixture
def db_conn():
    """Open a real Supabase connection for the test, close it after."""
    import psycopg  # local import: skipped tests must not require it at collect time

    db_url = os.environ.get("SUPABASE_DB_URL")
    if not db_url:
        pytest.skip("SUPABASE_DB_URL not set")
    conn = psycopg.connect(**_parse_db_url(db_url))
    try:
        yield conn
    finally:
        conn.close()


@pytest.fixture
def clean_test_user(db_conn):
    """Wipe every row tied to the reserved fixture phone before AND after.

    Running twice (setup + teardown) guarantees a previous failed run can't
    poison the next attempt. The order respects FKs: dependent rows first,
    then ``users_gym_profile`` and ``users`` last. Kairos checkpointer
    state is also cleared so the agent re-runs onboarding from scratch.
    """

    def wipe():
        with db_conn.cursor() as cur:
            cur.execute(
                "SELECT user_id FROM users WHERE full_phone_number = %s",
                (TEST_PHONE,),
            )
            user_ids = [row[0] for row in cur.fetchall()]

            for uid in user_ids:
                for table in (
                    "set_values",
                    "workouts",
                    "user_weekly_schedule",
                    "pending_tasks",
                    "magic_links",
                    "draft_routines",
                    "users_plans",
                ):
                    cur.execute(
                        f"DELETE FROM {table} WHERE user_id = %s", (uid,)
                    )

            cur.execute(
                "DELETE FROM users_gym_profile WHERE whatsapp_id = %s",
                (int(TEST_PHONE),),
            )
            cur.execute(
                "DELETE FROM users WHERE full_phone_number = %s",
                (TEST_PHONE,),
            )
            threads = [f"case6_{TEST_PHONE}", f"kyc_{TEST_PHONE}"]
            for table in ("checkpoint_writes", "checkpoint_blobs", "checkpoints"):
                cur.execute(
                    f"DELETE FROM {table} WHERE thread_id = ANY(%s)",
                    (threads,),
                )
        db_conn.commit()

    wipe()
    yield
    wipe()


@pytest.mark.skip(
    reason=(
        "BUG-1 regression guard. Requires KAIROS_URL + WORKOUT_API_URL + "
        "SUPABASE_DB_URL pointing at a staging stack and the BUG-1 fix "
        "deployed (Dev A's PR #27). Remove this decorator once the squad "
        "wires up the e2e job."
    )
)
@pytest.mark.asyncio
async def test_swap_persists_through_approve(clean_test_user, db_conn):
    """If the user swaps an exercise on the draft, the saved workout MUST
    use the swapped exercise, not the original.

    This is the contract Dev A and Dev B shipped in PR #27 (finalize
    endpoint reads draft_data instead of letting the LLM regenerate).
    """
    import httpx  # local: keep collect-time clean for environments without httpx

    kairos_url = os.environ.get("KAIROS_URL")
    api_url = os.environ.get("WORKOUT_API_URL")
    if not (kairos_url and api_url):
        pytest.skip("KAIROS_URL and WORKOUT_API_URL must both be set")

    async with httpx.AsyncClient(timeout=180.0) as http:
        # 1. Compact onboarding via Kairos /api/v1/chat. Three turns is the
        # minimum that triggers profile creation + draft generation.
        last_response = ""
        for message in (
            "Hola, quiero empezar a entrenar",
            "Ganar masa muscular, más de 3 años, 4 días por semana, en gimnasio",
            "Ninguna lesión",
        ):
            r = await http.post(
                f"{kairos_url}/api/v1/chat",
                json={
                    "phone_number": TEST_PHONE,
                    "display_name": TEST_NAME,
                    "message": message,
                },
            )
            r.raise_for_status()
            last_response = r.json().get("response", "")

        match = re.search(r"/draft\?c=([a-f0-9]{6})", last_response)
        assert match, f"final response did not contain a draft code: {last_response!r}"
        code = match.group(1)

        # 2. Capture the original day-1, exercise-1 and its first alternative.
        r = await http.get(f"{api_url}/api/v1/drafts/{code}")
        r.raise_for_status()
        draft = r.json()["data"]
        first_exercise = draft["days"][0]["exercises"][0]
        original_id = first_exercise["exercise_id"]
        assert first_exercise["alternatives"], "draft has no alternatives to swap to"
        alternative_id = first_exercise["alternatives"][0]["exercise_id"]
        assert alternative_id != original_id, "alternative is the same as the original — fixture broken"

        # 3. Swap.
        r = await http.patch(
            f"{api_url}/api/v1/drafts/{code}/swap",
            json={
                "day_number": first_exercise.get("day_number", 1),
                "exercise_order": first_exercise["exercise_order"],
                "new_exercise_id": alternative_id,
            },
        )
        r.raise_for_status()

        # 4. Approve. The response should include the new magic_link_code
        # (BUG-6b contract); we assert on it as a bonus regression guard.
        r = await http.post(f"{api_url}/api/v1/drafts/{code}/approve")
        r.raise_for_status()
        approve_body = r.json().get("data", {})
        assert approve_body.get("plan_id"), "approve response missing plan_id"
        # MagicLinkCode is optional per the contract — if present it must
        # be a non-empty short code.
        magic_code = approve_body.get("magic_link_code", "")
        if magic_code:
            assert re.fullmatch(r"[A-Za-z0-9]{4,32}", magic_code), (
                f"magic_link_code shape unexpected: {magic_code!r}"
            )

    # 5. Assert the workouts table reflects the swap. This is the line
    # that fails pre-BUG-1: the LLM regenerated and the original came back.
    with db_conn.cursor() as cur:
        cur.execute(
            """
            SELECT w.exercise_id
              FROM workouts w
              JOIN users u USING (user_id)
             WHERE u.full_phone_number = %s
               AND w.week = 1
               AND w.exercise_order = 1
             ORDER BY w.day_name
             LIMIT 1
            """,
            (TEST_PHONE,),
        )
        row = cur.fetchone()
    assert row is not None, "no workouts row created — finalize never ran"
    saved_exercise_id = row[0]
    assert saved_exercise_id == alternative_id, (
        f"swap was NOT persisted through approve: "
        f"saved={saved_exercise_id!r} expected={alternative_id!r} "
        f"(original was {original_id!r}). This is BUG-1 returning."
    )
