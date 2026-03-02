#!/usr/bin/env python3
import argparse
import json
import re
import sys
import urllib.parse
import urllib.request
from pathlib import Path

DEFAULT_JOBS = [
    "gateway",
    "query",
    "alerts",
    "governance",
    "aggregator",
    "features",
    "inference",
    "simulator",
]


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="Assert Prometheus up==1 for target jobs")
    p.add_argument("prom", nargs="?", default="http://localhost:9090", help="Prometheus base URL")
    p.add_argument("jobs", nargs="*", help="Explicit job names to check (fallback)")
    p.add_argument("--from-config", dest="from_config", help="Parse job_name values from prometheus.yml")
    return p.parse_args()


def parse_jobs_from_config(path: str) -> list[str]:
    text = Path(path).read_text(encoding="utf-8")
    # Supports e.g. job_name: foo, job_name: "foo", job_name: 'foo'
    pattern = re.compile(r"^\s*-?\s*job_name\s*:\s*(?:\"([^\"]+)\"|'([^']+)'|([^\s#]+))", re.MULTILINE)
    jobs: list[str] = []
    seen: set[str] = set()
    for m in pattern.finditer(text):
        job = (m.group(1) or m.group(2) or m.group(3) or "").strip()
        if job and job not in seen:
            seen.add(job)
            jobs.append(job)
    return jobs


def query_prom(prom_url: str, query: str) -> dict:
    q = urllib.parse.urlencode({"query": query}).encode()
    url = f"{prom_url}/api/v1/query?{q.decode()}"
    with urllib.request.urlopen(url, timeout=5) as r:
        return json.loads(r.read().decode())


def assert_job_up(prom_url: str, job: str) -> tuple[bool, str]:
    expr = f'up{{job="{job}"}}'
    data = query_prom(prom_url, expr)
    if data.get("status") != "success":
        return False, f"query failed for {job}: {data}"

    series = data.get("data", {}).get("result", [])
    if not series:
        return False, f"missing series for {job}"

    bad_vals = []
    for s in series:
        try:
            val = float(s.get("value", [None, "0"])[1])
        except (TypeError, ValueError, IndexError):
            val = 0.0
        if val < 1.0:
            bad_vals.append(val)

    if bad_vals:
        return False, f"down samples for {job}: {bad_vals}"

    return True, "ok"


def main() -> int:
    args = parse_args()

    config_jobs: list[str] = []
    if args.from_config:
        config_jobs = parse_jobs_from_config(args.from_config)
        if not config_jobs and not args.jobs:
            print(f"FAIL: no job_name entries found in {args.from_config}")
            return 1

    expected = config_jobs or args.jobs or DEFAULT_JOBS

    failures: list[str] = []
    for job in expected:
        ok, msg = assert_job_up(args.prom, job)
        if not ok:
            failures.append(msg)

    if failures:
        print("FAIL: Prometheus up check")
        for f in failures:
            print(" ", f)
        print("  Checked jobs:", expected)
        return 1

    source = f"config:{args.from_config}" if config_jobs else "explicit/default"
    print(f"PASS: Prometheus up==1 for ({source}): {expected}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
