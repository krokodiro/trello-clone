/** Server-side URL for the Go API (read at request time). */
export function getBackendApiUrl(): string {
  const publicUrl = process.env.API_PUBLIC_URL?.trim();
  if (publicUrl) {
    return publicUrl.startsWith("http") ? publicUrl.replace(/\/$/, "") : `https://${publicUrl.replace(/\/$/, "")}`;
  }
  const raw = process.env.API_URL?.trim();
  if (!raw) return "http://localhost:8080";
  if (raw.startsWith("http://") || raw.startsWith("https://")) {
    return raw.replace(/\/$/, "");
  }
  return `http://${raw}`;
}
