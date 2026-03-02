from __future__ import annotations

import hashlib
import json
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


def stable_json_dumps(payload: Any) -> str:
    return json.dumps(payload, sort_keys=True, separators=(",", ":"), default=str)


def stable_sha256(parts: list[str]) -> str:
    joined = "||".join(parts)
    return hashlib.sha256(joined.encode("utf-8")).hexdigest()


def snapshot_hash(payload: dict) -> str:
    ordered = {k: payload[k] for k in sorted(payload.keys())}
    return hashlib.sha256(stable_json_dumps(ordered).encode("utf-8")).hexdigest()


def read_json_file(path: str) -> dict[str, Any]:
    p = Path(path)
    if not path or not p.exists() or p.is_dir():
        return {}
    with p.open("r", encoding="utf-8") as f:
        data = json.load(f)
    return data if isinstance(data, dict) else {}


def parse_event_time(raw: Any) -> datetime:
    if isinstance(raw, datetime):
        if raw.tzinfo is None:
            return raw.replace(tzinfo=timezone.utc)
        return raw.astimezone(timezone.utc)
    if isinstance(raw, str) and raw.strip():
        txt = raw.strip()
        if txt.endswith("Z"):
            txt = txt[:-1] + "+00:00"
        try:
            parsed = datetime.fromisoformat(txt)
            if parsed.tzinfo is None:
                parsed = parsed.replace(tzinfo=timezone.utc)
            return parsed.astimezone(timezone.utc)
        except ValueError:
            pass
    return datetime.now(timezone.utc)


def bucket_rfc3339(raw: Any) -> str:
    bucket = parse_event_time(raw).replace(microsecond=0)
    return bucket.isoformat().replace("+00:00", "Z")
