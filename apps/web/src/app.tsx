import { Router } from "@solidjs/router";
import { FileRoutes } from "@solidjs/start/router";
import { Suspense } from "solid-js";
import { appBasePath } from "./lib/paths";
import "./app.css";

export default function App() {
  return <Router base={appBasePath} root={(props) => <Suspense>{props.children as any}</Suspense>}><FileRoutes /></Router>;
}
