import { Router } from "@solidjs/router";
import { FileRoutes } from "@solidjs/start/router";
import { Suspense } from "solid-js";
import { getRequestEvent } from "solid-js/web";
import { appBasePath } from "./lib/paths";
import "./app.css";

export default function App() {
  const requestURL = typeof window === "undefined" ? getRequestEvent()?.request.url : undefined;
  const routerURL = requestURL
    ? (() => {
        const url = new URL(requestURL);
        url.pathname = `${appBasePath}${url.pathname}`;
        return `${url.pathname}${url.search}${url.hash}`;
      })()
    : undefined;
  return <Router base={appBasePath} url={routerURL} root={(props) => <Suspense>{props.children as any}</Suspense>}><FileRoutes /></Router>;
}
