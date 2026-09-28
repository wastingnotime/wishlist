import { getRequestEvent } from "solid-js/web";
import { appPath } from "./paths";

export class ApiError extends Error {
  constructor(message: string, readonly status: number) {
    super(message);
    this.name = "ApiError";
  }
}

export async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const event = getRequestEvent();
  const origin = typeof window === "undefined"
    ? new URL(event?.request.url ?? "http://127.0.0.1:5173").origin
    : "";
  const response = await fetch(`${origin}${appPath(`/api/${path}`)}`, { ...init, cache: "no-store", headers: { ...(init?.body ? { "content-type": "application/json" } : {}), ...init?.headers } });
  if (!response.ok) {
    throw new ApiError(`Wishlist data could not be loaded (${response.status}).`, response.status);
  }
  if (response.status === 204) return undefined as T;
  return response.json() as Promise<T>;
}
