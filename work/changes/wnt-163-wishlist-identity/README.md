# WNT-163 — Wishlist identity adoption

Tracking: https://linear.app/wastingnotime/issue/WNT-163/align-wishlist-branding-and-parent-attribution-with-the-wnt-kit

## Change

Adopts Forward W **v1.0.0** from `wnt-design/assets/identity/v1.0.0/exports`.
The five supplied exports are copied byte-for-byte and served locally at
`/identity/v1.0.0/` beneath the configured deployment base. Export SHA-256 values
and the Wishlist social artwork hashes are in
[`manifest.json`](../../../apps/web/public/identity/v1.0.0/manifest.json).
The social SVG is a product-owned composition of the unchanged approved mark
and live Wishlist-specific text, rendered with `rsvg-convert`.

The board, privacy and admin headers now pair the mark with the Wishlist
descriptor. Each page has a same-tab parent signature with the accessible name
“made by WNT — Wasting No Time home”. The link has underlined text, a 24px mark,
clear space, a 32px minimum height and a 2px keyboard focus outline. Mint
`#9cffaa` on the darkest surface is above 14:1; muted text now uses `#929c95`
(above 6:1 against the product's dark panels). Board layout, app/lifecycle tabs,
voting, suggestions and authentication behavior remain intact. Ranking remains
a demand signal and never changes lifecycle state automatically.

Document metadata explicitly declares the supplied favicon variants, Apple
icon and a Wishlist social PNG. Asset links include `SERVER_BASE_URL`; social
images use the public `https://wastingnotime.org` origin plus that base. No
Community Lab metadata or parent-brand social card is reused.

## Evidence — 2026-10-15

Fetched deployed `https://wastingnotime.org/wishlist` before changes with curl:
title `WNT Wishlist — Public demand board`, description and theme color were
present; explicit favicon, Apple icon, OG and Twitter image declarations were
absent. This is the prior deployed baseline, not evidence of this change deployed.

Validation:

- Go API tests, TypeScript, seven auth/origin tests and root web build passed.
- `make test` reached browser startup but found port 18080 already occupied.
  Re-ran the unchanged browser suite using API 28080 / web 25173 against the
  disposable PostgreSQL test database: **all five passed** (board filters,
  empty state, admin publishing, OTP/vote/suggestion, full-document sign-in).
- Built with `WISHLIST_BASE_PATH=/wishlist` and served the production output
  on port 23000 with the real local API and disposable test data.
- `node scripts/identity-check.mjs` from `apps/web`: passed `/wishlist`,
  `/wishlist/`, privacy and admin at 1440×1000 and 390×844. Checks asset HTTP
  responses, loaded header/footer marks, metadata paths, accessible link names,
  Tab/Shift+Tab focus, Enter parent-home activation and same-tab navigation,
  horizontal fit, and desktop 200% zoom. Desktop/mobile screenshots are in
  [`screenshots/`](screenshots/). Browser accessible names were checked through
  role locators; no separate assistive-technology session was performed.
- Desktop board, mobile board and mobile admin screenshots visually inspected.

## Delivery boundary

This record proves local source and built-output behavior. Merge, candidate
publication and infra-platform promotion are separate. WNT-166 must verify the
final deployed revision, both entry URLs, asset responses and screenshots.
Rollback restores the previous source revision; the versioned kit is immutable.

## CI follow-up — candidate navigation selector

[Run 38825878647](https://github.com/wastingnotime/wishlist/actions/runs/38825878647)
failed because the built-image browser script still looked for the admin return
link named `Public Wishlist`; the updated header names it `Public board`.
Updated the exact role/name selector to match the rendered link. The existing
base-path URL assertions and publication/persistence checks remain in place.
`make integration` passed both phases locally with rebuilt API/web images,
including navigation through privacy/admin and persistence after API restart.
