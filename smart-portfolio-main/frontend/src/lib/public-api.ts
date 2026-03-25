const PUBLIC_API_URL = (import.meta.env.PUBLIC_API_URL || "").trim();

export function apiUrl(path: string): string {
  if (!path.startsWith("/")) {
    path = `/${path}`;
  }

  if (!PUBLIC_API_URL) {
    return path;
  }

  return `${PUBLIC_API_URL.replace(/\/+$/, "")}${path}`;
}
