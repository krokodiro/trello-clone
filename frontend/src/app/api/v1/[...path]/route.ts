import { NextRequest, NextResponse } from "next/server";
import { getBackendApiUrl } from "@/lib/api-backend-url";

type RouteContext = { params: Promise<{ path: string[] }> };

async function proxy(request: NextRequest, context: RouteContext) {
  const { path } = await context.params;
  const backend = getBackendApiUrl();
  const target = `${backend}/api/v1/${path.join("/")}${request.nextUrl.search}`;

  const headers = new Headers(request.headers);
  headers.delete("host");

  let body: ArrayBuffer | undefined;
  if (request.method !== "GET" && request.method !== "HEAD") {
    body = await request.arrayBuffer();
  }

  const res = await fetch(target, {
    method: request.method,
    headers,
    body,
  });
  return new NextResponse(res.body, {
    status: res.status,
    statusText: res.statusText,
    headers: res.headers,
  });
}

export const GET = proxy;
export const POST = proxy;
export const PUT = proxy;
export const PATCH = proxy;
export const DELETE = proxy;
