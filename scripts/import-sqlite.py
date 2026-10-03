#!/usr/bin/env python3
"""One-time import of the old local SQLite data into the local PostgreSQL container."""

import sqlite3
import subprocess
import sys
from pathlib import Path


TABLES = (
    "apps", "features", "identities", "otp_challenges", "otp_requests",
    "sessions", "votes", "suggestions",
)


def literal(value):
    if value is None:
        return "NULL"
    if isinstance(value, bool):
        return "TRUE" if value else "FALSE"
    if isinstance(value, int):
        return str(value)
    return "'" + str(value).replace("'", "''") + "'"


def main():
    source = Path(sys.argv[1] if len(sys.argv) > 1 else "apps/api/wishlist.db").resolve()
    if not source.is_file():
        raise SystemExit(f"SQLite source does not exist: {source}")
    source_db = sqlite3.connect(f"file:{source}?mode=ro", uri=True)
    snapshot = sqlite3.connect(":memory:")
    source_db.backup(snapshot)
    source_db.close()

    sql = ["\\set ON_ERROR_STOP on", "BEGIN;"]
    counts = {}
    for table in TABLES:
        cursor = snapshot.execute(f"SELECT * FROM {table}")
        columns = [item[0] for item in cursor.description]
        rows = cursor.fetchall()
        counts[table] = len(rows)
        for row in rows:
            values = dict(zip(columns, row))
            if table == "apps":
                values["active"] = bool(values["active"])
            if table == "identities":
                values["last_verified_at"] = values["created_at"]
            names = ",".join(values)
            data = ",".join(literal(value) for value in values.values())
            sql.append(f"INSERT INTO {table} ({names}) VALUES ({data}) ON CONFLICT DO NOTHING;")
    # A session is issued at verification and expires 30 days later. Preserve
    # that newer verification when importing an established SQLite identity.
    sql.append("""UPDATE identities i SET last_verified_at = GREATEST(
        i.last_verified_at,
        COALESCE((SELECT MAX(s.expires_at - INTERVAL '30 days')
                  FROM sessions s WHERE s.identity_id = i.id), i.last_verified_at)
    );""")
    sql.append("COMMIT;")
    command = ["docker", "compose", "exec", "-T", "postgres", "psql", "-q", "-U", "wishlist", "-d", "wishlist"]
    result = subprocess.run(command, input="\n".join(sql) + "\n", text=True, capture_output=True)
    if result.returncode:
        raise SystemExit(f"Import failed; PostgreSQL was rolled back. {result.stderr.strip()}")
    print("Copied missing rows from local SQLite snapshot:", ", ".join(f"{table}={count}" for table, count in counts.items()))
    print("The SQLite source was left untouched.")


if __name__ == "__main__":
    main()
