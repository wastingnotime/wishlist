import { A, useLocation, useNavigate } from "@solidjs/router";
import { For, Show, createMemo, createResource } from "solid-js";
import { ApiError, api } from "../lib/api";

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
  const selection = createMemo(() => {
    const query = new URLSearchParams(location.search);
    return { view: safeView(query.get("view") ?? undefined), app: query.get("app") ?? "" };
  });
  const [apps] = createResource(async () => {
    const result = await api<{ apps: AppChoice[] }>("apps");
    return result.apps;
  });
  const [board, { refetch }] = createResource(selection, async ({ view, app }) => {
    const query = new URLSearchParams({ view });
    if (app) query.set("app", app);
    return (await api<{ features: Feature[] }>(`features?${query.toString()}`)).features;
  });

  const chooseApp = (slug: string) => navigate(boardURL(selection().view, slug));
  const failure = () => board.error ?? apps.error;
  const errorMessage = () => {
    const error = failure();
    if (!error) return "";
    if (error instanceof ApiError && error.status === 503) return "The Wishlist service is taking a break. Try refreshing in a moment.";
    return "Wishlist data could not be loaded. Try refreshing.";
  };

  return (
    <div class="site-shell">
      <header class="topbar">
        <a class="brand" href="/" aria-label="Wasting No Time Wishlist home"><span class="brand-mark">W</span><span>WASTING NO TIME <b>/</b> WISHLIST</span></a>
        <span class="public-label"><span aria-hidden="true" class="online-dot" /> PUBLIC DEMAND BOARD</span>
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
        </section>

        <section class="board" aria-label="Wishlist board">
          <div class="board-toolbar">
            <label class="app-filter"><span>APP</span>
              <select aria-label="Filter by app" value={selection().app} onChange={(event) => chooseApp(event.currentTarget.value)}>
                <option value="">All apps</option>
                <For each={apps()}>{(app) => <option value={app.slug}>{app.name}</option>}</For>
              </select>
              <span class="select-chevron" aria-hidden="true">⌄</span>
            </label>
            <button class="refresh-button" onClick={() => refetch()} disabled={board.loading} aria-label="Refresh wishlist">
              <span aria-hidden="true" classList={{ spinning: board.loading }}>↻</span><span>Refresh</span>
            </button>
          </div>

          <nav class="view-tabs" aria-label="Feature status">
            <For each={views}>{(view) => <A class="view-tab" classList={{ active: selection().view === view }} href={boardURL(view, selection().app)} aria-current={selection().view === view ? "page" : undefined}>
              <span>{viewLabel[view]}</span><span class={`tab-indicator ${view}`} aria-hidden="true" />
            </A>}</For>
          </nav>

          <div class="board-heading"><div><p class="eyebrow">{viewLabel[selection().view]} board</p><h2>{selection().view === "voting" ? "Community requests" : selection().view === "producing" ? "In the workshop" : "Out in the world"}</h2></div>
            <span class="result-count">{board()?.length ?? 0} {board()?.length === 1 ? "feature" : "features"}</span>
          </div>

          <Show when={failure()}><div class="error-banner" role="alert"><span>{errorMessage()}</span><button onClick={() => { void refetch(); }} class="retry-button">Try again</button></div></Show>
          <Show when={!board.loading && !failure()}>
            <Show when={(board()?.length ?? 0) > 0} fallback={<div class="empty-state"><span aria-hidden="true">◇</span><h3>Nothing here yet</h3><p>There are no features in this view for the selected app.</p></div>}>
              <div class="feature-list" aria-live="polite">
                <For each={board()}>{(feature, index) => <article class="feature-card" data-feature-slug={feature.slug}>
                  <div class="rank-number">{String(index() + 1).padStart(2, "0")}</div>
                  <div class="vote-count" aria-label={`${feature.vote_count} votes`}><span aria-hidden="true">▲</span><strong>{feature.vote_count}</strong><small>VOTES</small></div>
                  <div class="feature-copy"><div class="feature-meta"><span class={`status-pill ${feature.status}`}><i aria-hidden="true" />{viewLabel[feature.status]}</span><span class="app-name">{feature.app_name}</span></div>
                    <h3>{feature.title}</h3><p>{feature.description}</p>
                    <Show when={feature.status === "delivered" && feature.delivered_at}><time class="delivered-date" dateTime={feature.delivered_at!}>Shipped {dateLabel(feature.delivered_at)}</time></Show>
                  </div>
                  <Show when={feature.status === "delivered" && feature.delivery_url}><a class="delivery-link" href={feature.delivery_url!} target="_blank" rel="noreferrer">View release <span aria-hidden="true">↗</span></a></Show>
                </article>}</For>
              </div>
            </Show>
          </Show>
          <Show when={board.loading}><div class="loading-state" role="status"><span class="loader" />Loading the board…</div></Show>
        </section>
      </main>

      <footer class="footer"><span>WNT <b>·</b> WISHLIST</span><p>Community signal <span>→</span> WNT judgment <span>→</span> shipped work</p><small>Ideas are proposals. Votes are not promises.</small></footer>
    </div>
  );
}
