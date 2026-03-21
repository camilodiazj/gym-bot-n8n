"""Shared checkpointer factory — PostgresSaver for production, InMemorySaver for dev.

Uses a connection pool (AsyncConnectionPool) to handle Cloud Run idle timeouts
and automatic reconnection.

When SUPABASE_DB_URL is set, uses AsyncPostgresSaver (persistent).
Otherwise falls back to InMemorySaver (dev/testing).
"""

import os
from contextlib import asynccontextmanager

from dotenv import load_dotenv
from langgraph.checkpoint.memory import InMemorySaver

load_dotenv()


def _parse_db_url(db_url: str) -> str:
    """Parse a PostgreSQL URL into a conninfo string.

    Handles usernames with dots (postgres.project-ref) and passwords
    with special characters (@, #, etc.) that break urlparse.
    """
    rest = db_url.split("://", 1)[1]
    creds, hostpart = rest.rsplit("@", 1)
    user, password = creds.split(":", 1)
    host_port, dbname = hostpart.split("/", 1)
    host, port = host_port.rsplit(":", 1)

    escaped_password = password.replace("'", "\\'")
    return f"host={host} port={port} dbname={dbname} user={user} password='{escaped_password}'"


@asynccontextmanager
async def postgres_checkpointer_context():
    """Async context manager that yields a PostgresSaver with a connection pool.

    The pool automatically handles reconnection when connections drop
    (e.g., Cloud Run idle timeouts, Supabase pooler resets).
    """
    db_url = os.getenv("SUPABASE_DB_URL", "")

    if db_url:
        from psycopg_pool import AsyncConnectionPool
        from psycopg.rows import dict_row
        from langgraph.checkpoint.postgres.aio import AsyncPostgresSaver

        conninfo = _parse_db_url(db_url)

        pool = AsyncConnectionPool(
            conninfo=conninfo,
            min_size=1,
            max_size=5,
            kwargs={"autocommit": True, "prepare_threshold": 0, "row_factory": dict_row},
        )
        await pool.open()

        checkpointer = AsyncPostgresSaver(pool)

        try:
            await checkpointer.setup()
        except Exception:
            pass  # Tables already created

        try:
            yield checkpointer
        finally:
            await pool.close()
            return

    yield InMemorySaver()
