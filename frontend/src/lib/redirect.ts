export function getRedirectPath(search?: string): string | null {
  const params = new URLSearchParams(search ?? (typeof window !== "undefined" ? window.location.search : ""));
  const redirect = params.get("redirect");
  if (!redirect || !redirect.startsWith("/") || redirect.startsWith("//")) {
    return null;
  }
  return redirect;
}

export function withRedirect(path: string, redirect: string | null): string {
  if (!redirect) return path;
  const sep = path.includes("?") ? "&" : "?";
  return `${path}${sep}redirect=${encodeURIComponent(redirect)}`;
}
