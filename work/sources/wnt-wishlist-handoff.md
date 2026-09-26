# WastingNoTime Wishlist — Codex Handoff

## 1. Governing sentence

**WNT Wishlist is a public demand signal for WastingNoTime apps: people can propose and vote for features, while WNT retains the decision of what to build and when.**

Votes express demand. They are **not promises** and do not automatically determine the development roadmap.

## 2. Product goal

Build a small public web app where:

- anyone can browse apps, features, votes, and delivery status;
- visitors can vote after verifying an email with OTP;
- visitors can suggest features after email verification;
- suggestions are private until an admin reviews and publishes them;
- admins can create apps and features directly;
- admins move features through `voting -> producing -> delivered`;
- voting features are primarily ranked by vote count;
- every public view can be filtered by app.

The product should remain intentionally small. Do not turn it into a social network, issue tracker, or full product-management suite.

## 3. Public information architecture

The main public interface has three views/tabs:

1. **Voting** — published features currently accepting votes.
2. **Producing** — features WNT has selected and is currently building.
3. **Delivered** — features that have shipped.

All three views are public and support filtering by app.

Suggested features awaiting moderation are **not public**.

Example:

```text
WASTINGNOTIME / WISHLIST

[All apps v]        [Voting] [Producing] [Delivered]

▲ 184  Family sharing                 Cat Care
▲ 137  Export health history          Cat Care
▲  82  Medication reminders           Cat Care
▲  41  Dark mode                      Sliding Tasks

[Suggest a feature]
```

## 4. Identity model

Do **not** require traditional user registration for the MVP.

Use lightweight verified identities:

```text
visitor -> email -> OTP -> verified identity/session
```

A visitor can browse anonymously. Verification is requested only when they attempt an identity-sensitive action such as voting or suggesting a feature.

After successful OTP verification, create/reuse an identity associated with the normalized email and establish a reasonably long-lived authenticated session.

Do not expose password creation, profiles, usernames, password reset, account pages, avatars, or social features.

### Privacy

Never expose voter email addresses publicly. Public vote data is aggregate only.

## 5. Voting

A verified identity can have at most one active vote per feature.

Expected behavior:

- anonymous visitor clicks Vote;
- if not verified, request email and OTP;
- after verification, register the intended vote;
- clicking an already-voted feature may remove the vote (toggle behavior);
- vote count updates accordingly;
- duplicate votes from the same identity must be prevented server-side.

The database constraint, not only application code, should enforce uniqueness of `(identity_id, feature_id)`.

### Ranking

For MVP, rank `voting` features primarily by:

```text
vote_count DESC
```

Use a deterministic secondary ordering such as publication date / ID.

Do not implement Reddit/Stack Exchange hot-ranking, time decay, reputation weighting, or algorithmic prioritization in v1.

Important semantic rule:

> Votes tell WNT what people want. They do not promise what WNT will build next.

An admin can move any feature to Producing regardless of its ranking.

## 6. Suggestions

Verified visitors can suggest a feature for an existing app.

Suggested minimum fields:

- app
- title
- optional short description

A submitted suggestion starts in a private moderation state such as `pending`.

Admin actions:

- publish/accept;
- edit before publishing;
- reject;
- merge into an existing feature where appropriate.

When accepted as a new feature, it becomes public with status `voting`.

Do not expose rejected or pending suggestions publicly.

## 7. Apps

An App is a WNT product/project for which features can be requested.

Minimum fields:

```text
id
slug
name
description? 
url?
active
created_at
updated_at
```

Examples might include Cat Care, Sliding Tasks, and Wishlist itself. Do not hard-code those examples as required seed data unless useful for local development.

Admins can create/edit apps. Public users can filter the three feature views by app.

## 8. Features

Suggested model:

```text
Feature
- id
- app_id
- slug
- title
- description?
- status: voting | producing | delivered
- published_at
- producing_at?
- delivered_at?
- delivery_url?
- created_at
- updated_at
```

`delivery_url` can point to the relevant release, GitHub artifact, WNT episode/post, or application page.

A feature moved from Voting to Producing or Delivered keeps its historical votes. Vote count must not reset when status changes.

## 9. Suggestions vs. features

Keep moderation separate from the public Feature lifecycle.

Possible model:

```text
Suggestion
- id
- app_id
- identity_id
- title
- description?
- status: pending | accepted | rejected | merged
- resulting_feature_id?
- created_at
- reviewed_at?
```

This avoids overloading `Feature.status` with moderation concerns.

## 10. OTP

OTP exists only to establish control of an email address and reduce trivial vote manipulation.

Requirements:

- short-lived OTP;
- single-use or invalidated after successful verification;
- expiration;
- rate limits for requesting and attempting OTPs;
- normalized email addresses;
- do not reveal whether an email already exists in the system;
- store OTP securely (prefer hashed/challenge representation rather than reusable plaintext);
- establish a session after successful verification.

Keep the provider behind an interface so local development can use a fake/logging implementation and production can use an email provider.

## 11. Admin

Admin functionality can be intentionally utilitarian. It does not need the same visual polish as the public site.

Admin must be able to:

- create/edit/deactivate apps;
- create/edit features;
- inspect pending suggestions;
- accept/edit/reject/merge suggestions;
- move a feature between `voting`, `producing`, and `delivered`;
- set a delivery URL when appropriate;
- inspect vote totals.

Admin authentication is separate from visitor OTP identity. Use the simplest secure mechanism consistent with the existing WNT infrastructure/codebase.

## 12. Public feature lifecycle

```text
Visitor suggestion
       |
       v
   [pending]          private moderation
       |
       | admin accepts
       v
    VOTING            public + votable
       |
       | admin selects
       v
   PRODUCING          public
       |
       | shipped
       v
   DELIVERED          public + delivery link
```

Admins can also create a feature directly in the public lifecycle without a visitor suggestion.

## 13. Domain invariants

Treat these as important business rules:

1. Only published features are visible publicly.
2. Pending/rejected suggestions are never public.
3. One verified identity has at most one vote per feature.
4. Votes belong to the feature and survive lifecycle transitions.
5. Only admins change a public feature's lifecycle status.
6. Visitor votes influence ordering but do not automatically change status.
7. Only admins can make visitor suggestions publicly votable.
8. Public APIs never expose email addresses or OTP material.
9. Delivered features may retain their final vote count as historical context.
10. App filtering must behave consistently across Voting, Producing, and Delivered.

## 14. MVP pages / routes

Exact routing is implementation-dependent, but the product needs these surfaces.

### Public

```text
/                       wishlist board; default Voting
/?app=<slug>&view=voting
/?app=<slug>&view=producing
/?app=<slug>&view=delivered
/feature/<slug-or-id>   optional if detail warrants a page
/suggest                may also be a modal/drawer
```

### Identity

OTP request/verification can be modal-driven or dedicated routes.

### Admin

```text
/admin
/admin/apps
/admin/features
/admin/suggestions
```

Do not create pages merely to satisfy this route sketch. Prefer the smallest coherent UX.

## 15. API/use cases

The architecture should expose business use cases rather than coupling rules to HTTP handlers.

Core operations:

```text
ListApps
ListFeatures(view, appFilter)
RequestOtp(email)
VerifyOtp(email, code)
VoteForFeature(identity, feature)
RemoveVote(identity, feature)
SubmitSuggestion(identity, app, title, description)

AdminCreateApp
AdminUpdateApp
AdminCreateFeature
AdminUpdateFeature
AdminChangeFeatureStatus
AdminListSuggestions
AdminAcceptSuggestion
AdminRejectSuggestion
AdminMergeSuggestion
```

Naming can follow the conventions of the chosen stack/codebase.

## 16. Suggested persistence model

Conceptually:

```text
apps
identities
otp_challenges / verification_challenges
sessions
features
votes
suggestions
admins / admin auth integration
```

Important indexes/constraints:

```text
apps.slug UNIQUE
identities.normalized_email UNIQUE
votes(identity_id, feature_id) UNIQUE
features(app_id, slug) UNIQUE   # if using per-app slugs
features(status)
features(app_id, status)
suggestions(status)
```

Optimize only when justified by observed usage. Correct invariants and simple queries matter more than premature scale work.

## 17. UI direction

Follow WNT's existing identity rather than inventing a generic SaaS dashboard:

- dark/minimal interface;
- restrained neon/cyberpunk accents where appropriate;
- high information density without clutter;
- clear typography;
- voting should feel immediate;
- status transitions should be visually obvious;
- responsive/mobile-friendly;
- accessible keyboard/focus states;
- no engagement gimmicks.

The board itself is the hero. Avoid dashboards full of vanity metrics.

A feature row/card should communicate roughly:

```text
▲ 137
Export health history
Cat Care
```

Producing may emphasize active work rather than vote controls. Delivered may show delivery date and a link to the result.

## 18. Explicit non-goals for v1

Do **not** add unless implementation constraints require them:

- user profiles;
- passwords;
- usernames/avatars;
- comments/discussions;
- replies;
- reputation or karma;
- downvotes;
- multiple vote weights;
- social following;
- notifications/subscriptions;
- public roadmap commitments;
- automatic promotion to Producing;
- AI prioritization;
- complex scoring/hotness algorithms;
- teams/organizations;
- payments;
- analytics dashboards;
- native mobile apps.

Keep scope aggressively small.

## 19. Abuse/security baseline

This is public input, so assume hostile requests even if expected traffic is small.

At minimum:

- validate and bound all text input;
- escape/safely render user content;
- CSRF protection where relevant to the chosen auth model;
- secure session cookies;
- OTP request/attempt throttling;
- suggestion throttling;
- server-side authorization for every admin operation;
- database uniqueness for votes;
- avoid exposing identity/email information;
- basic audit-friendly timestamps for moderation/lifecycle changes.

Do not build elaborate anti-fraud infrastructure in v1. OTP + rate limiting + database constraints are sufficient initially.

## 20. Observability / operations

Keep this proportional to the app.

Provide:

- liveness endpoint;
- readiness endpoint if the app depends on database/external services;
- structured logs;
- errors for failed OTP delivery/verification without leaking sensitive data;
- enough telemetry to diagnose failed votes, suggestions, and admin actions.

Integrate with existing WNT infrastructure conventions where they exist rather than creating a parallel operational stack.

## 21. Testing priorities

Prefer tests around invariants and use cases.

Must cover at least:

- anonymous users can browse public states;
- private suggestions never appear in public lists;
- OTP verification establishes an identity/session;
- invalid/expired OTP fails;
- one identity cannot vote twice for the same feature;
- vote can be removed if toggle behavior is implemented;
- vote count ordering works;
- app filter works for all three views;
- accepting a suggestion creates/publishes the correct feature;
- only admin can change lifecycle status;
- votes remain after `voting -> producing -> delivered`;
- delivered feature can expose its delivery URL;
- public responses do not leak emails.

## 22. MVP acceptance scenario

The MVP is successful when this complete path works:

1. Admin creates `Cat Care`.
2. Admin publishes two Cat Care features.
3. Anonymous visitor opens Wishlist and sees both under Voting.
4. Visitor votes for one feature.
5. App requests email verification.
6. Visitor receives/enters OTP.
7. Vote is registered and persists across reload/session.
8. Same identity cannot create a duplicate vote.
9. Visitor submits a new Cat Care suggestion.
10. Suggestion is invisible publicly.
11. Admin reviews and publishes it.
12. It appears in Voting with zero votes initially.
13. Votes determine ordering within Voting.
14. Admin moves one feature to Producing even if it is not #1.
15. Feature disappears from Voting and appears under Producing with its votes preserved.
16. Admin marks it Delivered and adds a delivery URL.
17. It appears publicly under Delivered with historical vote count and delivery link.

If this path is clean, deployable, tested, and understandable, v1 is done.

## 23. Implementation instruction for Codex

First inspect the repository and existing WNT conventions before choosing libraries, architecture, deployment, authentication integration, or directory structure.

Prefer existing conventions over introducing new technology.

Before coding:

1. summarize the current repository architecture;
2. identify reusable infrastructure/components;
3. propose the smallest implementation plan;
4. call out assumptions or conflicts with this specification;
5. then implement in vertical slices.

Suggested vertical slices:

```text
1. apps + public feature board
2. admin feature lifecycle
3. visitor OTP identity
4. voting
5. suggestions + moderation
6. delivered links/polish
7. operational hardening
```

For every slice, keep the application runnable and tests green.

Do not expand the product because a framework makes additional features easy.

## 24. Product principle

The system should remain a **sensor, not a governor**:

```text
community signal
       |
       v
    Wishlist
       |
       v
 WNT judgment
       |
       v
 development
       |
       v
 delivered artifact
```

The ranking makes demand visible. WNT retains autonomy over what is actually built.
