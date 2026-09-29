import { A, useLocation, useNavigate } from "@solidjs/router";
import { For, Show, createEffect, createMemo, createResource, createSignal, onMount } from "solid-js";
import { ApiError, api } from "../lib/api";
import { appPath } from "../lib/paths";

type View = "voting" | "producing" | "delivered";
type AppChoice = { id: string; slug: string; name: string; description: string; url: string };
type Feature = {
  id: string;
  slug: string;
  title: string;
  description: string;
  app_slug: string;
  app_name: string;
  status: View;
  vote_count: number;
  published_at: string;
  delivered_at: string | null;
  delivery_url: string | null;
};
const views: View[] = ["voting", "producing", "delivered"];
const viewLabel: Record<View, string> = { voting: "Voting", producing: "Producing", delivered: "Delivered" };

function safeView(raw: string | undefined): View {
  return views.includes(raw as View) ? raw as View : "voting";
}

function boardURL(view: View, app: string) {
  const query = new URLSearchParams({ view });
  if (app) query.set("app", app);
  return `/?${query.toString()}`;
}

function dateLabel(value: string | null) {
  if (!value) return "";
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(new Date(value));
}

export default function PublicBoard() {
  const location = useLocation();
  const navigate = useNavigate();
  const [session, setSession] = createSignal<{verified:boolean; vote_feature_ids:string[]}|null>(null);
  const [ready,setReady]=createSignal(false);
  const [sessionLoaded,setSessionLoaded]=createSignal(false);
  const [authEmail, setAuthEmail] = createSignal("");
  const [otpCode, setOtpCode] = createSignal("");
  const [authStep, setAuthStep] = createSignal<"closed"|"email"|"code">("closed");
  const [pendingVote, setPendingVote] = createSignal("");
  const [suggestOpen, setSuggestOpen] = createSignal(false);
  const [suggestApp, setSuggestApp] = createSignal("");
  const [suggestTitle, setSuggestTitle] = createSignal("");
  const [suggestDescription, setSuggestDescription] = createSignal("");
  const [notice, setNotice] = createSignal("");
  const [busy, setBusy] = createSignal(false);
  onMount(() => { setReady(true);void api<{verified:boolean;vote_feature_ids:string[]}>("session").then(setSession).catch(()=>setSession(null)).finally(()=>setSessionLoaded(true)); });
  const [apps] = createResource(async () => {
    const result = await api<{ apps: AppChoice[] }>("apps");
    return result.apps;
  });
  const selection = createMemo(() => {
    const query = new URLSearchParams(location.search);
    const requestedApp = query.get("app");
    const availableApps = apps() ?? [];
    return { view: safeView(query.get("view") ?? undefined), app: availableApps.find(app => app.slug === requestedApp)?.slug ?? availableApps[0]?.slug ?? "" };
  });
  const selectedApp = createMemo(() => apps()?.find(app => app.slug === selection().app));
  createEffect(() => {
    if (!ready() || !selectedApp()) return;
    const query = new URLSearchParams(location.search);
    if (query.get("app") !== selection().app || query.get("view") !== selection().view)
      navigate(boardURL(selection().view, selection().app), { replace: true });
  });
  const [board, { refetch }] = createResource(() => apps()?.length ? selection() : undefined, async ({ view, app }) => {
    const query = new URLSearchParams({ view });
    if (app) query.set("app", app);
    return (await api<{ features: Feature[] }>(`features?${query.toString()}`)).features;
  });

  const requireIdentity = (action: string) => { if (!session()?.verified) { setPendingVote(action); setAuthStep("email"); return false; } return true; };
  const toggleVote = async (feature: Feature) => {
    if (!requireIdentity(feature.id)) return;
    setBusy(true); try { await api(`features/${feature.id}/vote`, {method:"POST",body:"{}"}); const s=await api<{verified:boolean;vote_feature_ids:string[]}>("session"); setSession(s); await refetch(); }
    catch { setNotice("Your vote could not be saved. Please try again."); } finally { setBusy(false); }
  };
  const requestCode = async (event: SubmitEvent) => { event.preventDefault(); setBusy(true); try { await api("otp",{method:"POST",body:JSON.stringify({email:authEmail()})}); setAuthStep("code"); } catch { setNotice("We could not send a sign-in code. Check the email address and try again."); } finally { setBusy(false); } };
  const verifyCode = async (event: SubmitEvent) => { event.preventDefault(); setBusy(true); try { await api("otp/verify",{method:"POST",body:JSON.stringify({email:authEmail(),code:otpCode()})}); const s=await api<{verified:boolean;vote_feature_ids:string[]}>("session");setSession(s);setAuthStep("closed");const action=pendingVote();setPendingVote("");if(action==="suggest") setSuggestOpen(true); else if(action) await toggleVoteById(action); } catch { setNotice("That code is invalid or expired. Request a new one and try again."); } finally { setBusy(false); } };
  const toggleVoteById = async (id:string) => { setBusy(true); try { await api(`features/${id}/vote`,{method:"POST",body:"{}"});setSession(await api<{verified:boolean;vote_feature_ids:string[]}>("session"));await refetch(); }catch{setNotice("Your vote could not be saved. Please try again.");}finally{setBusy(false);} };
  const submitSuggestion = async (event: SubmitEvent) => { event.preventDefault(); if(!requireIdentity("suggest")) return; setBusy(true);try{await api("suggestions",{method:"POST",body:JSON.stringify({app_id:suggestApp(),title:suggestTitle(),description:suggestDescription()})});setSuggestOpen(false);setSuggestTitle("");setSuggestDescription("");setNotice("Thanks. Your suggestion is private and will be reviewed by WNT.");}catch{setNotice("Your suggestion could not be submitted. Please try again.");}finally{setBusy(false);} };
  const failure = () => board.error ?? apps.error;
  const errorMessage = () => {
    const error = failure();
    if (!error) return "";
    if (error instanceof ApiError && error.status === 503) return "The Wishlist service is taking a break. Try refreshing in a moment.";
    return "Wishlist data could not be loaded. Try refreshing.";
  };

  return (
    <div class="site-shell">
      <aside class="side-label side-label-left" aria-hidden="true">wnt // wishlist</aside>
      <aside class="side-label side-label-right" aria-hidden="true">ideas in motion</aside>
      <header class="topbar">
        <a class="brand" href={appPath("/")} aria-label="Wasting No Time Wishlist home"><span class="brand-mark">wl</span><span>wnt / wishlist</span></a>
          <div class="header-actions"><span class="public-label"><span aria-hidden="true" class="online-dot" /> PUBLIC DEMAND BOARD</span><button class="suggest-button" disabled={!sessionLoaded()} onClick={()=>{setSuggestApp(selectedApp()?.id??"");setSuggestOpen(true);}}>Suggest an idea</button><A class="admin-nav-link" href="/admin">Admin</A><Show when={session()?.verified}><button class="signout-button" disabled={!sessionLoaded()} onClick={async()=>{await api("session",{method:"DELETE"});setSession(null);}}>Sign out</button></Show></div>
      </header>

      <main>
        <section class="hero" aria-labelledby="page-title">
          <div class="hero-copy">
            <p class="eyebrow">Ideas become visible here</p>
            <h1 id="page-title">What should we<br /><span>make better?</span></h1>
            <p class="intro">See what people want across WNT apps, what is in progress, and what has shipped.</p>
          </div>
          <aside class="principle" aria-label="Wishlist principle">
            <span class="principle-icon" aria-hidden="true">↗</span>
            <p>Votes show demand.<br /><strong>WNT decides what to build.</strong></p>
          </aside>
          <div class="orbit orbit-one" aria-hidden="true" /><div class="orbit orbit-two" aria-hidden="true" />
        </section>

        <section class="board" aria-label="Wishlist board">
          <div class="board-toolbar">
            <span class="app-selector-label">Browse an app</span>
            <button class="refresh-button" onClick={() => refetch()} disabled={board.loading} aria-label="Refresh wishlist">
              <span aria-hidden="true" classList={{ spinning: board.loading }}>↻</span><span>Refresh</span>
            </button>
          </div>
          <nav class="app-choices" aria-label="Choose an app">
            <For each={apps()}>{app => <A class="app-choice" classList={{ active: selection().app === app.slug }} href={boardURL(selection().view, app.slug)} aria-current={selection().app === app.slug ? "page" : undefined}>{app.name}</A>}</For>
          </nav>

          <nav class="view-tabs" aria-label="Feature status">
            <For each={views}>{(view) => <A class="view-tab" classList={{ active: selection().view === view }} href={boardURL(view, selection().app)} aria-current={selection().view === view ? "page" : undefined}>
              <span>{viewLabel[view]}</span><span class={`tab-indicator ${view}`} aria-hidden="true" />
            </A>}</For>
          </nav>

          <div class="board-heading"><div><p class="eyebrow">{viewLabel[selection().view]} board</p><h2>{selectedApp()?.name ?? "Wishlist"}</h2><p class="board-subtitle">{selection().view === "voting" ? "Community requests" : selection().view === "producing" ? "In the workshop" : "Out in the world"}</p></div>
            <span class="result-count">{board()?.length ?? 0} {board()?.length === 1 ? "feature" : "features"}</span>
          </div>

          <Show when={failure()}><div class="error-banner" role="alert"><span>{errorMessage()}</span><button onClick={() => { void refetch(); }} class="retry-button">Try again</button></div></Show>
          <Show when={!board.loading && !failure()}>
            <Show when={(board()?.length ?? 0) > 0} fallback={<div class="empty-state"><span aria-hidden="true">◇</span><h3>Nothing here yet</h3><p>{selectedApp() ? "There are no features in this view for the selected app." : "There are no apps available yet."}</p></div>}>
              <div class="feature-list" aria-live="polite">
                <For each={board()}>{(feature, index) => <article class="feature-card" data-feature-slug={feature.slug}>
                  <div class="rank-number">{String(index() + 1).padStart(2, "0")}</div>
                  <div class="vote-count" aria-label={`${feature.vote_count} votes`}><span aria-hidden="true">▲</span><strong>{feature.vote_count}</strong><small>VOTES</small></div>
                  <div class="feature-copy"><div class="feature-meta"><span class={`status-pill ${feature.status}`}><i aria-hidden="true" />{viewLabel[feature.status]}</span></div>
                    <h3>{feature.title}</h3><p>{feature.description}</p>
                    <Show when={feature.status === "delivered" && feature.delivered_at}><time class="delivered-date" dateTime={feature.delivered_at!}>Shipped {dateLabel(feature.delivered_at)}</time></Show>
                  </div>
                  <Show when={feature.status === "voting"}><button class="vote-button" aria-pressed={session()?.vote_feature_ids.includes(feature.id)??false} disabled={!sessionLoaded()||busy()} onClick={()=>void toggleVote(feature)}>{session()?.vote_feature_ids.includes(feature.id) ? "Voted ✓" : "Vote ↑"}</button></Show>
                  <Show when={feature.status === "delivered" && feature.delivery_url}><a class="delivery-link" href={feature.delivery_url!} target="_blank" rel="noreferrer">View release <span aria-hidden="true">↗</span></a></Show>
                </article>}</For>
              </div>
            </Show>
          </Show>
          <Show when={board.loading}><div class="loading-state" role="status"><span class="loader" />Loading the board…</div></Show>
        </section>
      </main>

      <Show when={notice()}><div class="toast" role="status">{notice()}<button onClick={()=>setNotice("")} aria-label="Dismiss">×</button></div></Show>
      <Show when={authStep()!=="closed"}><div class="modal-backdrop"><section class="modal" role="dialog" aria-modal="true" aria-labelledby="auth-title"><button class="modal-close" onClick={()=>setAuthStep("closed")} aria-label="Close">×</button><p class="eyebrow">Verified visitors</p><h2 id="auth-title">{authStep()==="email"?"Sign in to vote":"Check your inbox"}</h2><p>{authStep()==="email"?"We’ll email you a one-time code. Your address stays private.":`Enter the six-digit code sent to ${authEmail()}.`}</p><Show when={authStep()==="email"} fallback={<form onSubmit={verifyCode}><label>One-time code<input required inputmode="numeric" pattern="[0-9]{6}" autocomplete="one-time-code" value={otpCode()} onInput={e=>setOtpCode(e.currentTarget.value)}/></label><button class="primary-button" disabled={busy()}>Verify and continue</button><button type="button" class="text-button" onClick={()=>setAuthStep("email")}>Use a different email</button></form>}><form onSubmit={requestCode}><label>Email address<input type="email" required autocomplete="email" value={authEmail()} onInput={e=>setAuthEmail(e.currentTarget.value)}/></label><button class="primary-button" disabled={busy()}>Send sign-in code</button></form></Show></section></div></Show>
      <Show when={suggestOpen()}><div class="modal-backdrop"><section class="modal" role="dialog" aria-modal="true" aria-labelledby="suggest-title"><button class="modal-close" onClick={()=>setSuggestOpen(false)} aria-label="Close">×</button><p class="eyebrow">Private until reviewed</p><h2 id="suggest-title">Suggest an idea</h2><p>Suggestions are visible only to WNT reviewers until approved.</p><form onSubmit={submitSuggestion}><label>WNT app<select required value={suggestApp()} onChange={e=>setSuggestApp(e.currentTarget.value)}><For each={apps()}>{a=><option value={a.id}>{a.name}</option>}</For></select></label><label>Idea title<input required maxlength="160" value={suggestTitle()} onInput={e=>setSuggestTitle(e.currentTarget.value)}/></label><label>What would this help you do?<textarea maxlength="2000" rows="4" value={suggestDescription()} onInput={e=>setSuggestDescription(e.currentTarget.value)}/></label><button class="primary-button" disabled={busy()}>Continue</button></form></section></div></Show>

      <footer class="footer"><span>WNT <b>·</b> WISHLIST</span><p>Community signal <span>→</span> WNT judgment <span>→</span> shipped work</p><small>Ideas are proposals. Votes are not promises.</small></footer>
    </div>
  );
}
