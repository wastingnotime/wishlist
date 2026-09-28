import { defineConfig } from "@solidjs/start/config";

const configuredBasePath = process.env.WISHLIST_BASE_PATH?.trim() ?? "";
if (configuredBasePath && (!configuredBasePath.startsWith("/") || /[?#]/.test(configuredBasePath))) {
  throw new Error("WISHLIST_BASE_PATH must be an absolute URL path");
}

const baseURL = configuredBasePath
  ? `/${configuredBasePath.split("/").filter(Boolean).join("/")}`
  : "";

export default defineConfig({ server: { baseURL } });
