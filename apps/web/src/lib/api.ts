import { getRequestEvent } from "solid-js/web";

export class ApiError extends Error {
  constructor(message: string, readonly status: number) {
    super(message);
    this.name = "ApiError";
  }
}

export async function api<T>(path: string): Promise<T> {
  const event = getRequestEvent();
  const origin = typeof window === "undefined"
    ? new URL(event?.request.url ?? "http://127.0.0.1:5173").origin
    : "";
  const response = await fetch(`${origin}/api/${path}`, { cache: "no-store" });
  if (!response.ok) {
    throw new ApiError(`Wishlist data could not be loaded (${response.status}).`, response.status);
  }
  return response.json() as Promise<T>;
}
