import { A } from "@solidjs/router";
import { For, Show, createSignal, onMount } from "solid-js";
import { ApiError, api } from "../lib/api";
import { appPath } from "../lib/paths";

type PrivacyData = {
  email: string;
  created_at: string;
  last_verified_at: string;
  active_sessions: number;
  votes: { feature_id: string; app_name: string; title: string; status: string; created_at: string }[];
  suggestions: { id: string; app_name: string; title: string; description: string; status: string; created_at: string; reviewed_at: string | null; resulting_feature_id: string | null }[];
};

function date(value: string) {
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(new Date(value));
}

export default function Privacy() {
  const [state, setState] = createSignal<"loading" | "verify" | "ready" | "erased" | "error">("loading");
  const [data, setData] = createSignal<PrivacyData | null>(null);
  const [email, setEmail] = createSignal("");
  const [code, setCode] = createSignal("");
  const [verificationStep, setVerificationStep] = createSignal<"email" | "code">("email");
  const [newEmail, setNewEmail] = createSignal("");
  const [changeCode, setChangeCode] = createSignal("");
  const [changeStep, setChangeStep] = createSignal<"email" | "code">("email");
  const [message, setMessage] = createSignal("");
  const [busy, setBusy] = createSignal(false);

  const load = async () => {
    try {
      const result = await api<PrivacyData>("privacy/data");
      setData(result);
      setEmail(result.email);
      setState("ready");
      setMessage("");
    } catch (error) {
      if (error instanceof ApiError && (error.status === 401 || error.status === 403)) {
        setState("verify");
        setMessage(error.status === 403 ? "Verify your email again to open your private records." : "Verify your email to view your Wishlist records.");
      } else {
        setState("error");
        setMessage("Your records could not be loaded. Please try again.");
      }
    }
  };
  onMount(() => { void load(); });

  const requestCode = async (event: SubmitEvent) => {
    event.preventDefault(); setBusy(true); setMessage("");
    try {
      await api("otp", { method: "POST", body: JSON.stringify({ email: email() }) });
      setVerificationStep("code");
      setMessage("We sent a six-digit code. It expires in ten minutes.");
    } catch { setMessage("The code could not be sent. Check the address and try again."); }
    finally { setBusy(false); }
  };
  const verify = async (event: SubmitEvent) => {
    event.preventDefault(); setBusy(true); setMessage("");
    try {
      await api("otp/verify", { method: "POST", body: JSON.stringify({ email: email(), code: code() }) });
      setCode(""); setVerificationStep("email");
      await load();
    } catch { setMessage("That code is invalid or expired. Request a new one and try again."); }
    finally { setBusy(false); }
  };
  const requestChange = async (event: SubmitEvent) => {
    event.preventDefault(); setBusy(true); setMessage("");
    try {
      await api("privacy/email/request", { method: "POST", body: JSON.stringify({ new_email: newEmail() }) });
      setChangeStep("code");
      setMessage("Enter the code sent to your new address.");
    } catch (error) {
      if (error instanceof ApiError && error.status === 403) { setState("verify"); setMessage("Verify your current email again before changing it."); }
      else setMessage("The new-address code could not be sent. Check the address and try again.");
    } finally { setBusy(false); }
  };
  const completeChange = async (event: SubmitEvent) => {
    event.preventDefault(); setBusy(true); setMessage("");
    try {
      await api("privacy/email/verify", { method: "POST", body: JSON.stringify({ new_email: newEmail(), code: changeCode() }) });
      setChangeCode(""); setNewEmail(""); setChangeStep("email");
      await load();
      setMessage("Your email address was updated. Other Wishlist sessions were signed out.");
    } catch (error) {
      if (error instanceof ApiError && error.status === 409) setMessage("That address is already used. Contact privacy@wastingnotime.org for help.");
      else if (error instanceof ApiError && error.status === 403) { setState("verify"); setMessage("Verify your current email again before changing it."); }
      else setMessage("That code is invalid or expired. Request a new one and try again.");
    } finally { setBusy(false); }
  };
  const erase = async () => {
    if (!window.confirm("Delete your Wishlist email, votes, private suggestions, and sessions? This cannot be undone.")) return;
    setBusy(true); setMessage("");
    try {
      await api("privacy/data", { method: "DELETE" });
      setData(null); setState("erased");
    } catch (error) {
      if (error instanceof ApiError && error.status === 403) { setState("verify"); setMessage("Verify your email again before deleting your records."); }
      else setMessage("Your records could not be deleted. Contact privacy@wastingnotime.org for help.");
    } finally { setBusy(false); }
  };

  return <div class="site-shell privacy-shell">
    <header class="topbar"><a class="brand" href={appPath("/")} aria-label="Wasting No Time Wishlist home"><span class="brand-mark">wl</span><span>wnt / wishlist</span></a><A class="admin-nav-link" href="/">Back to board</A></header>
    <main class="privacy-main">
      <p class="eyebrow">Wishlist visitor data</p><h1>Your privacy controls</h1>
      <p class="privacy-intro">See the email, votes, and private suggestions linked to your Wishlist account. Email verification protects these actions.</p>
      <Show when={message()}><p class="privacy-message" role="status">{message()}</p></Show>
      <Show when={state() === "loading"}><p>Loading your records…</p></Show>
      <Show when={state() === "error"}><button class="primary-button" onClick={() => void load()}>Try again</button></Show>
      <Show when={state() === "verify"}>
        <section class="privacy-panel"><h2>Verify your email</h2><p>We will send a one-time code to the address linked to your Wishlist records.</p>
          <Show when={verificationStep() === "email"} fallback={<form onSubmit={verify}><label>Six-digit code<input required inputmode="numeric" pattern="[0-9]{6}" autocomplete="one-time-code" value={code()} onInput={event => setCode(event.currentTarget.value)} /></label><button class="primary-button" disabled={busy()}>Open my records</button><button type="button" class="text-button" onClick={() => setVerificationStep("email")}>Use a different email</button></form>}>
            <form onSubmit={requestCode}><label>Email address<input type="email" required autocomplete="email" value={email()} onInput={event => setEmail(event.currentTarget.value)} /></label><button class="primary-button" disabled={busy()}>Send code</button></form>
          </Show>
        </section>
      </Show>
      <Show when={state() === "ready" && data()}>{records => <>
        <section class="privacy-panel"><h2>Account</h2><dl><div><dt>Email</dt><dd>{records().email}</dd></div><div><dt>Created</dt><dd>{date(records().created_at)}</dd></div><div><dt>Last verified</dt><dd>{date(records().last_verified_at)}</dd></div><div><dt>Active sessions</dt><dd>{records().active_sessions}</dd></div></dl></section>
        <section class="privacy-panel"><h2>Your votes</h2><Show when={records().votes.length} fallback={<p>No votes are linked to this address.</p>}><ul><For each={records().votes}>{vote => <li><strong>{vote.title}</strong><span>{vote.app_name} · {vote.status} · {date(vote.created_at)}</span></li>}</For></ul></Show></section>
        <section class="privacy-panel"><h2>Your private suggestions</h2><Show when={records().suggestions.length} fallback={<p>No private suggestions are linked to this address.</p>}><ul><For each={records().suggestions}>{suggestion => <li><strong>{suggestion.title}</strong><span>{suggestion.app_name} · {suggestion.status} · {date(suggestion.created_at)}</span><p>{suggestion.description || "No description provided."}</p></li>}</For></ul></Show></section>
        <section class="privacy-panel"><h2>Correct your email</h2><p>We will verify the new address. Other Wishlist sessions will be signed out.</p><Show when={changeStep() === "email"} fallback={<form onSubmit={completeChange}><label>Code sent to {newEmail()}<input required inputmode="numeric" pattern="[0-9]{6}" autocomplete="one-time-code" value={changeCode()} onInput={event => setChangeCode(event.currentTarget.value)} /></label><button class="primary-button" disabled={busy()}>Update email</button><button type="button" class="text-button" onClick={() => setChangeStep("email")}>Use another address</button></form>}><form onSubmit={requestChange}><label>New email address<input type="email" required autocomplete="email" value={newEmail()} onInput={event => setNewEmail(event.currentTarget.value)} /></label><button class="primary-button" disabled={busy()}>Send verification code</button></form></Show></section>
        <section class="privacy-panel privacy-danger"><h2>Delete your Wishlist records</h2><p>This removes your email, votes, private suggestions, and active sessions from the live Wishlist database. Vote totals will update. Published feature text is reviewed separately if it may identify you.</p><button class="privacy-delete" disabled={busy()} onClick={() => void erase()}>Delete my Wishlist records</button></section>
      </>}</Show>
      <Show when={state() === "erased"}><section class="privacy-panel"><h2>Records deleted</h2><p>Your linked records were removed from the live Wishlist database and you were signed out. A future sign-in creates a new identity. If a published feature contains personal details, contact us for review.</p><A class="admin-nav-link" href="/">Return to board</A></section></Show>
      <p class="privacy-contact">For help, requests involving public feature text, or records outside the live Wishlist database, email <a href="mailto:privacy@wastingnotime.org">privacy@wastingnotime.org</a>.</p>
    </main>
  </div>;
}
