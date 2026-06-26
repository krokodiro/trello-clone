import { getServerAppConfig } from "@/lib/runtime-config";

export async function GET() {
  const config = getServerAppConfig();
  return Response.json({
    apiUrl: config.apiUrl || null,
    wsUrl: config.wsUrl || null,
    mode: config.apiUrl ? "direct" : "proxy",
  });
}
