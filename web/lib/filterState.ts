export type GlobalFilters = {
  window?: "1h" | "24h" | "7d" | "30d" | "90d" | "1y" | "5y" | "20y" | "custom";
  start?: string;
  end?: string;
  from?: string;
  to?: string;
  countryCode?: string;
  region?: string;
  sector?: string;
  industry?: string;
  venue?: string;
  symbol?: string;
  locale?: string;
  limit?: number;
  q?: string;
};

export const WINDOW_OPTIONS: GlobalFilters["window"][] = ["1h", "24h", "7d", "30d", "90d", "1y", "5y", "20y", "custom"];

const FILTER_KEYS: Array<keyof GlobalFilters> = [
  "window",
  "start",
  "end",
  "countryCode",
  "region",
  "sector",
  "industry",
  "venue",
  "symbol",
  "locale",
  "limit",
  "q",
];

function normalizeText(value: unknown): string {
  return String(value ?? "").trim();
}

export function normalizeWindow(value: unknown): GlobalFilters["window"] {
  const candidate = normalizeText(value) as GlobalFilters["window"];
  return WINDOW_OPTIONS.includes(candidate) ? candidate : "24h";
}

type ParamsLike = URLSearchParams | { entries(): IterableIterator<[string, string]> }

export function parseFilterState(source?: ParamsLike | string | null): GlobalFilters {
  const params =
    typeof source === "string"
      ? new URLSearchParams(source.startsWith("?") ? source.slice(1) : source)
      : new URLSearchParams(source ? Array.from(source.entries()) : []);

  const out: GlobalFilters = { window: normalizeWindow(params.get("window")) };
  const start = normalizeText(params.get("start") || params.get("from"));
  const end = normalizeText(params.get("end") || params.get("to"));
  if (start) out.start = start;
  if (end) out.end = end;
  for (const key of FILTER_KEYS) {
    if (key === "window" || key === "start" || key === "end") {
      continue;
    }
    const raw = params.get(key);
    if (!raw) {
      continue;
    }
    if (key === "limit") {
      const parsed = Number.parseInt(raw, 10);
      if (Number.isFinite(parsed) && parsed > 0) {
        out.limit = parsed;
      }
      continue;
    }
    out[key] = normalizeText(raw) as never;
  }
  if (out.start || out.end) {
    out.window = "custom";
  }
  return out;
}

export function mergeFilterState(current: GlobalFilters, patch: Partial<GlobalFilters>): GlobalFilters {
  const next: GlobalFilters = { ...current, ...patch };
  if (next.from && !next.start) next.start = next.from;
  if (next.to && !next.end) next.end = next.to;
  delete next.from;
  delete next.to;
  const requestedWindow = Object.prototype.hasOwnProperty.call(patch, "window") ? normalizeWindow(patch.window) : undefined;
  if (requestedWindow && requestedWindow !== "custom") {
    delete next.start;
    delete next.end;
  } else if (normalizeText(next.start) || normalizeText(next.end)) {
    next.window = "custom";
  }
  next.window = normalizeWindow(next.window);
  for (const key of FILTER_KEYS) {
    if (key === "limit") {
      if (!next.limit || next.limit <= 0) {
        delete next.limit;
      }
      continue;
    }
    if (key === "window") {
      continue;
    }
    const value = normalizeText(next[key]);
    if (!value) {
      delete next[key];
      continue;
    }
    next[key] = value as never;
  }
  if (next.window !== "custom") {
    delete next.start;
    delete next.end;
  }
  return next;
}

export function filtersToQuery(filters: GlobalFilters = {}) {
  const normalized = mergeFilterState({ window: "24h" }, filters);
  const p = new URLSearchParams();
  for (const key of FILTER_KEYS) {
    if (key === "window" && normalized.window === "custom" && (normalized.start || normalized.end)) {
      continue;
    }
    const value = normalized[key];
    if (value === undefined || value === null || value === "") {
      continue;
    }
    p.set(key, String(value));
  }
  const q = p.toString();
  return q ? `?${q}` : "";
}

export function toDateTimeLocalValue(value?: string): string {
  const raw = normalizeText(value);
  if (!raw) {
    return "";
  }
  const date = new Date(raw);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000);
  return local.toISOString().slice(0, 16);
}

export function fromDateTimeLocalValue(value: string): string {
  const raw = normalizeText(value);
  if (!raw) {
    return "";
  }
  const local = new Date(raw);
  if (Number.isNaN(local.getTime())) {
    return "";
  }
  return local.toISOString();
}
