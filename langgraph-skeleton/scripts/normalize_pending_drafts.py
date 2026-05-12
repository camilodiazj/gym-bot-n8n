"""One-shot migration to normalize existing draft_routines.draft_data rows.

Validates each pending draft against the canonical Pydantic schema and rewrites
the JSONB with the normalized representation (rir coerced to string, alternative
`link` renamed to `video_link`, unknown fields dropped).

Idempotent: re-running on already-normalized rows is a no-op write.

Usage:
    cd langgraph-skeleton
    .venv/bin/python -m scripts.normalize_pending_drafts
"""

from __future__ import annotations

import json
import os
import re
import sys
from pathlib import Path

import psycopg2
import psycopg2.extras
from pydantic import ValidationError

# Make sure repo root is on sys.path so `from cases.case6_unified_agent...` works
ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT))

from cases.case6_unified_agent.draft_schema import normalize_draft_data  # noqa: E402


def _parse_db_url(raw: str) -> dict:
    """psycopg2 chokes when the password contains '@'. Parse manually."""
    m = re.match(r"^postgresql://([^:]+):(.+)@([^@/]+):(\d+)/(.+)$", raw)
    if not m:
        raise ValueError(f"unrecognized SUPABASE_DB_URL format")
    user, password, host, port, db = m.groups()
    return dict(user=user, password=password, host=host, port=int(port), dbname=db, sslmode="require")


def main() -> int:
    db_url = os.environ.get("SUPABASE_DB_URL")
    if not db_url:
        print("ERROR: SUPABASE_DB_URL not set", file=sys.stderr)
        return 2

    conn = psycopg2.connect(**_parse_db_url(db_url))
    conn.autocommit = False

    fixed = 0
    unchanged = 0
    failed = 0

    try:
        with conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor) as cur:
            cur.execute(
                "SELECT code, draft_data FROM draft_routines WHERE status = 'pending' ORDER BY created_at"
            )
            rows = cur.fetchall()

            print(f"Found {len(rows)} pending draft(s)")
            for row in rows:
                code = row["code"]
                raw = row["draft_data"]
                try:
                    normalized = normalize_draft_data(raw)
                except ValidationError as e:
                    failed += 1
                    print(f"  [SKIP] {code}: validation failed — {e.error_count()} issue(s)")
                    continue

                if normalized == raw:
                    unchanged += 1
                    print(f"  [OK]   {code}: already normalized")
                    continue

                with conn.cursor() as write_cur:
                    write_cur.execute(
                        "UPDATE draft_routines SET draft_data = %s::jsonb WHERE code = %s",
                        (json.dumps(normalized), code),
                    )
                fixed += 1
                print(f"  [FIX]  {code}: normalized")

        conn.commit()
    except Exception:
        conn.rollback()
        raise
    finally:
        conn.close()

    print(f"\nSummary: fixed={fixed} unchanged={unchanged} failed={failed}")
    return 0 if failed == 0 else 1


if __name__ == "__main__":
    raise SystemExit(main())
