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
  if (typeof window !== "undefined" && window.__APP_CONFIG__) {
    return window.__APP_CONFIG__;
  }
  return {
    apiUrl: cleanUrl(process.env.NEXT_PUBLIC_API_URL || ""),
    wsUrl: cleanUrl(process.env.NEXT_PUBLIC_WS_URL || ""),
  };
}

function cleanUrl(value: string) {
  return value.trim().replace(/\/$/, "");
}
