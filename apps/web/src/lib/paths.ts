const configuredBasePath = import.meta.env?.SERVER_BASE_URL ?? "";

export const appBasePath = configuredBasePath.replace(/\/+$/, "");

export function appPath(path: string): string {
  const routePath = path.startsWith("/") ? path : `/${path}`;
  return `${appBasePath}${routePath}` || "/";
}
