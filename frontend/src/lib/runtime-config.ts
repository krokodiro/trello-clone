export type AppConfig = {
  apiUrl: string;
  wsUrl: string;
};

declare global {
  interface Window {
    __APP_CONFIG__?: AppConfig;
  }
}

export function getServerAppConfig(): AppConfig {
  return {
    apiUrl: cleanUrl(
      process.env.API_PUBLIC_URL || process.env.NEXT_PUBLIC_API_URL || ""
    ),
    wsUrl: cleanUrl(
      process.env.WS_PUBLIC_URL || process.env.NEXT_PUBLIC_WS_URL || ""
    ),
  };
}

export function getClientConfig(): AppConfig {
  if (typeof window !== "undefined") {
    if (window.__APP_CONFIG__?.apiUrl) {
      return window.__APP_CONFIG__;
    }
    const fromDom = readConfigFromDom();
    if (fromDom.apiUrl) {
      window.__APP_CONFIG__ = fromDom;
      return fromDom;
    }
  }
  return {
    apiUrl: cleanUrl(process.env.NEXT_PUBLIC_API_URL || ""),
    wsUrl: cleanUrl(process.env.NEXT_PUBLIC_WS_URL || ""),
  };
}

function readConfigFromDom(): AppConfig {
  if (typeof document === "undefined") {
    return { apiUrl: "", wsUrl: "" };
  }
  const el = document.getElementById("app-config");
  if (!el) return { apiUrl: "", wsUrl: "" };
  return {
    apiUrl: cleanUrl(el.getAttribute("data-api-url") || ""),
    wsUrl: cleanUrl(el.getAttribute("data-ws-url") || ""),
  };
}

function cleanUrl(value: string) {
  return value.trim().replace(/\/$/, "");
}

let configPromise: Promise<AppConfig> | null = null;

/** Load API URL from /api/config if not already available (client only). */
export async function loadClientConfig(): Promise<AppConfig> {
  const existing = getClientConfig();
  if (existing.apiUrl) return existing;

  if (typeof window === "undefined") return existing;

  if (!configPromise) {
    configPromise = fetch("/api/config")
      .then((r) => r.json())
      .then((data: { apiUrl?: string | null; wsUrl?: string | null }) => {
        const cfg: AppConfig = {
          apiUrl: cleanUrl(data.apiUrl || ""),
          wsUrl: cleanUrl(data.wsUrl || ""),
        };
        window.__APP_CONFIG__ = cfg;
        return cfg;
      })
      .catch(() => existing);
  }
  return configPromise;
}
