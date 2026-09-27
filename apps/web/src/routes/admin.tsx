import { For, Show, createResource, createSignal, onMount } from "solid-js";
import { api } from "../lib/api";

type App = { id: string; slug: string; name: string; description: string; url: string; active: boolean };
type Suggestion = { id: string; app_id: string; title: string; description: string; created_at: string };
type Feature = { id: string; title: string; description: string; app_slug: string; app_name: string; delivery_url: string | null; status: "voting" | "producing" | "delivered"; vote_count: number };
type Status = Feature["status"];

export default function Admin() {
  const [token, setToken] = createSignal("");
  const [notice, setNotice] = createSignal("");
  const [refresh, setRefresh] = createSignal(0);
  const [ready, setReady] = createSignal(false);
  const [adminSession, setAdminSession] = createSignal<{ mode: "loading" | "token" | "casdoor"; authorized: boolean }>({ mode: "loading", authorized: false });
  onMount(() => {
    setReady(true);
    void fetch("/auth/admin-session", { cache: "no-store" })
      .then(async response => { if (!response.ok) throw new Error("Admin login is unavailable."); return response.json(); })
      .then(setAdminSession)
      .catch(() => setNotice("Admin login is unavailable. Refresh and try again."));
  });
  const auth = (): Record<string, string> => adminSession().mode === "casdoor" ? {} : { authorization: `Bearer ${token()}` };
  const accessKey = () => adminSession().mode === "casdoor" ? (adminSession().authorized ? "casdoor" : "") : (adminSession().mode === "token" ? token() : "");
  const [apps] = createResource(() => [refresh(), accessKey()] as const, async ([, key]) => key ? (await api<{ apps: App[] }>("admin/apps", { headers: auth() })).apps : []);
  const [suggestions] = createResource(() => [refresh(), accessKey()] as const, async ([, key]) => key ? (await api<{ suggestions: Suggestion[] }>("admin/suggestions", { headers: auth() })).suggestions : []);
  const [features] = createResource(() => [refresh(), accessKey()] as const, async ([, key]) => {
    if (!key) return [];
    const groups = await Promise.all(((["voting", "producing", "delivered"] as const).map(async view => (await api<{ features: Feature[] }>(`features?view=${view}`)).features)));
    return groups.flat();
  });
  const run = async (action: () => Promise<unknown>, message: string) => {
    try { await action(); setNotice(message); setRefresh(value => value + 1); }
    catch { setNotice(adminSession().mode === "casdoor" ? "The action failed. Check your admin access and entered values." : "The action failed. Check the admin token and entered values."); }
  };
  const createApp = async (event: SubmitEvent) => {
    event.preventDefault(); const htmlForm=event.currentTarget as HTMLFormElement;const form = new FormData(htmlForm);
    await run(() => api("admin/apps", { method: "POST", headers: auth(), body: JSON.stringify({ slug: form.get("slug"), name: form.get("name"), description: form.get("description"), url: form.get("url") }) }), "App created.");
    htmlForm.reset();
  };
  const createFeature = async (event: SubmitEvent) => {
    event.preventDefault(); const htmlForm=event.currentTarget as HTMLFormElement;const form = new FormData(htmlForm);
    await run(() => api("admin/features", { method: "POST", headers: auth(), body: JSON.stringify({ app_id: form.get("app_id"), slug: form.get("slug"), title: form.get("title"), description: form.get("description") }) }), "Feature published to Voting.");
    htmlForm.reset();
  };
  const acceptSuggestion = async (suggestion: Suggestion, edit: boolean) => {
    const title=edit?window.prompt("Feature title",suggestion.title):suggestion.title;if(!title)return;
    const description=edit?window.prompt("Description",suggestion.description):suggestion.description;if(description===null)return;
    const slug=title.toLowerCase().normalize("NFKD").replace(/[^a-z0-9]+/g,"-").replace(/^-|-$/g,"").slice(0,80);
    await run(() => api(`admin/suggestions/${suggestion.id}/accept`, { method: "POST", headers: auth(), body: JSON.stringify({ slug, title, description }) }), "Suggestion published to Voting.");
  };
  const rejectSuggestion=async(suggestion:Suggestion)=>run(()=>api(`admin/suggestions/${suggestion.id}/reject`,{method:"POST",headers:auth(),body:"{}"}),"Suggestion rejected.");
  const mergeSuggestion=async(suggestion:Suggestion)=>{const slug=apps()?.find(app=>app.id===suggestion.app_id)?.slug;const candidates=features()?.filter(feature=>feature.app_slug===slug)??[];if(!candidates.length){setNotice("No existing feature is available to merge into for this app.");return;}const selection=window.prompt(`Enter the feature ID to merge into:\n${candidates.map(feature=>`${feature.id} — ${feature.title}`).join("\n")}`);if(!selection)return;await run(()=>api(`admin/suggestions/${suggestion.id}/merge`,{method:"POST",headers:auth(),body:JSON.stringify({feature_id:selection.trim()})}),"Suggestion merged into the selected feature.");};
  const editApp = async (app: App) => {
    const name = window.prompt("App name", app.name); if (!name) return;
    const url = window.prompt("App URL", app.url); if (url === null) return;
    await run(() => api(`admin/apps/${app.id}`, { method: "PATCH", headers: auth(), body: JSON.stringify({ name, description: app.description, url, active: app.active }) }), "App updated.");
  };
  const toggleApp = async (app: App) => {
    if (app.active && !window.confirm(`Deactivate ${app.name}? Existing published features remain visible.`)) return;
    await run(() => api(`admin/apps/${app.id}`, { method: "PATCH", headers: auth(), body: JSON.stringify({ name: app.name, description: app.description, url: app.url, active: !app.active }) }), app.active ? "App deactivated." : "App reactivated.");
  };
  const editFeature = async (feature: Feature) => {
    const title = window.prompt("Feature title", feature.title); if (!title) return;
    const description = window.prompt("Description", feature.description); if (description === null) return;
    await run(() => api(`admin/features/${feature.id}`, { method: "PATCH", headers: auth(), body: JSON.stringify({ title, description, status: feature.status, delivery_url: feature.delivery_url ?? "" }) }), "Feature updated.");
  };
  const changeStatus = async (feature: Feature, status: Status) => {
    let url = "";
    if (status === "delivered") { url = window.prompt("Delivery URL (https://…)") ?? ""; }
    await run(() => api(`admin/features/${feature.id}`, { method: "PATCH", headers: auth(), body: JSON.stringify({ title: feature.title, description: feature.description, status, delivery_url: url }) }), `Feature moved to ${status}.`);
  };

  return <main class="admin-shell">
    <header class="admin-head"><a href="/">← Public Wishlist</a><span>WNT / ADMIN</span></header>
    <h1>Manage the wishlist</h1>
    <p class="admin-intro">Admin actions decide what becomes public and what moves into production. Votes remain a signal.</p>
    <Show when={adminSession().mode === "token"}><label class="admin-token">Admin access token<input type="password" autocomplete="off" disabled={!ready()} value={token()} onInput={event => setToken(event.currentTarget.value)} placeholder="Enter WISHLIST_ADMIN_TOKEN" /></label></Show>
    <Show when={adminSession().mode === "casdoor" && adminSession().authorized}><p class="admin-intro">Signed in with Casdoor. <a class="admin-auth-link" href="/auth/logout">Sign out</a></p></Show>
    <Show when={adminSession().mode === "casdoor" && !adminSession().authorized}><div class="admin-login"><p>{new URLSearchParams(typeof window === "undefined" ? "" : window.location.search).get("error") === "forbidden" ? "This Casdoor account is not authorized to manage the Wishlist." : "Sign in with an authorized WNT account to manage the Wishlist."}</p><a class="admin-auth-link" href="/auth/login">Sign in with Casdoor →</a></div></Show>
    <Show when={notice()}><p role="status" class="admin-notice">{notice()}</p></Show>
    <Show when={accessKey()} fallback={<Show when={adminSession().mode === "token"}><p class="admin-intro">Enter the admin token to manage apps, features, and suggestions.</p></Show>}>
      <div class="admin-grid">
        <section class="admin-panel"><h2>Apps</h2>
          <form onSubmit={createApp}><label>Slug<input name="slug" required pattern="[a-z0-9]+(-[a-z0-9]+)*" /></label><label>Name<input name="name" required maxlength="100" /></label><label>Description<input name="description" maxlength="500" /></label><label>URL<input name="url" type="url" /></label><button>Create app</button></form>
          <h3>Existing apps</h3><ul><For each={apps()}>{app => <li><span>{app.name}<code>{app.slug}</code></span><span class="admin-inline-actions"><button onClick={() => void editApp(app)}>Edit</button><button class="admin-secondary" onClick={() => void toggleApp(app)}>{app.active ? "Deactivate" : "Reactivate"}</button></span></li>}</For></ul>
        </section>
        <section class="admin-panel"><h2>Publish a feature</h2>
          <form onSubmit={createFeature}><label>App<select name="app_id" required><For each={apps()?.filter(app => app.active)}>{app => <option value={app.id}>{app.name}</option>}</For></select></label><label>Feature slug<input name="slug" required pattern="[a-z0-9]+(-[a-z0-9]+)*" /></label><label>Title<input name="title" required maxlength="160" /></label><label>Description<textarea name="description" maxlength="2000" rows="3" /></label><button>Publish to Voting</button></form>
        </section>
      </div>
      <section class="admin-panel admin-wide"><h2>Pending suggestions <span>{suggestions()?.length ?? 0}</span></h2>
        <Show when={(suggestions()?.length ?? 0) > 0} fallback={<p>No pending suggestions.</p>}><For each={suggestions()}>{suggestion => <article class="admin-suggestion"><small>{apps()?.find(app => app.id === suggestion.app_id)?.name ?? "WNT app"} · {new Date(suggestion.created_at).toLocaleDateString()}</small><h3>{suggestion.title}</h3><p>{suggestion.description || "No description provided."}</p><div><button onClick={() => void acceptSuggestion(suggestion,false)}>Accept and publish</button><button class="admin-secondary" onClick={() => void acceptSuggestion(suggestion,true)}>Edit and publish</button><button class="admin-secondary" onClick={() => void mergeSuggestion(suggestion)}>Merge into feature</button><button class="admin-secondary" onClick={() => void rejectSuggestion(suggestion)}>Reject</button></div></article>}</For></Show>
      </section>
      <section class="admin-panel admin-wide"><h2>Feature lifecycle</h2><div class="admin-feature-list"><For each={features()}>{feature => <article><div><strong>{feature.title}</strong><small>{feature.app_name} · {feature.vote_count} votes · {feature.status}</small></div><div class="lifecycle-actions"><button class="admin-secondary" onClick={() => void editFeature(feature)}>Edit</button><Show when={feature.status === "voting"}><button onClick={() => void changeStatus(feature, "producing")}>Move to Producing</button></Show><Show when={feature.status === "producing"}><button onClick={() => void changeStatus(feature, "delivered")}>Mark Delivered</button></Show><Show when={feature.status === "delivered"}><span>Delivered</span></Show></div></article>}</For></div></section>
    </Show>
  </main>;
}
