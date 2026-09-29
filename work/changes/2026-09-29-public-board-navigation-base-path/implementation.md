# Public board navigation under the Wishlist base path

## Problem

Tracked as [Wishlist issue #12](https://github.com/wastingnotime/wishlist/issues/12).

The production view tabs generated paths such as
`/wishlist/wishlist/?view=producing&app=sliding-tasks`. The board URL helper
included the `/wishlist` deployment prefix, then Solid Router applied its own
configured `/wishlist` base to the `<A>` link.

The app filter used the same prefixed URL but navigated with a full page load,
so its path construction also bypassed Router's base handling.

## Change

Build board navigation URLs as router-relative paths (`/?...`) and use the
router's `navigate` function for filter selection. Solid Router applies the
configured base once to both the view tabs and filter navigation. Native links
that are outside Router remain unchanged.

## Validation

- Reviewed all `boardURL` call sites in `apps/web/src/routes/index.tsx`.
- Local tests were not run in this session.
