/** Server-side URL for the Go API (read at request time, not build time). */
export function getBackendApiUrl(): string {
  const raw = process.env.API_URL?.trim();
  if (!raw) return "http://localhost:8080";
  if (raw.startsWith("http://") || raw.startsWith("https://")) {
    return raw.replace(/\/$/, "");
  }
  return `http://${raw}`;
}
