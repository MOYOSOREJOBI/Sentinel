import { NextRequest, NextResponse } from "next/server";

const MAP: Record<string, string> = {
  "gateway": "UPSTREAM_GATEWAY",
  "gateway-api": "UPSTREAM_GATEWAY",
  "query": "UPSTREAM_QUERY",
  "alerts": "UPSTREAM_ALERTS",
  "governance": "UPSTREAM_GOVERNANCE",
};

function upstreamFor(service: string) {
  const mode = process.env.SENTINEL_UPSTREAM_MODE || "local";
  if (mode === "compose") {
    const key = MAP[service];
    const v = key ? process.env[key] : "";
    if (!v) throw new Error(`Missing upstream env for service=${service} key=${key}`);
    return v.replace(/\/+$/, "");
  }
  if (service === "gateway" || service === "gateway-api") return "http://localhost:8080";
  if (service === "query") return "http://localhost:8085";
  if (service === "alerts") return "http://localhost:8083";
  if (service === "governance") return "http://localhost:8084";
  throw new Error(`Unknown service=${service}`);
}

async function proxy(req: NextRequest, service: string, path: string[]) {
  const base = upstreamFor(service);
  const url = new URL(`${base}/${path.join("/")}`);
  req.nextUrl.searchParams.forEach((v, k) => url.searchParams.set(k, v));

  const body = ["GET","HEAD"].includes(req.method) ? undefined : await req.text();
  const res = await fetch(url, { method: req.method, headers: req.headers, body, redirect: "manual" });

  const headers = new Headers(res.headers);
  const setCookie = res.headers.get("set-cookie");
  if (setCookie) headers.set("set-cookie", setCookie);

  if ([204,205,304].includes(res.status)) return new NextResponse(null, { status: res.status, headers });
  return new NextResponse(await res.arrayBuffer(), { status: res.status, headers });
}

export const dynamic = "force-dynamic";

export async function GET(req: NextRequest, ctx: any) { return proxy(req, ctx.params.service, ctx.params.path); }
export async function POST(req: NextRequest, ctx: any) { return proxy(req, ctx.params.service, ctx.params.path); }
export async function PUT(req: NextRequest, ctx: any) { return proxy(req, ctx.params.service, ctx.params.path); }
export async function PATCH(req: NextRequest, ctx: any) { return proxy(req, ctx.params.service, ctx.params.path); }
export async function DELETE(req: NextRequest, ctx: any) { return proxy(req, ctx.params.service, ctx.params.path); }
