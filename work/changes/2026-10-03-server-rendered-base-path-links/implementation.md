# Server-rendered Wishlist links keep the production base path

## Problem

In production, the initial HTML rendered internal links such as `/privacy`,
`/admin`, and `/?view=...` outside the `/wishlist` mount. Clicking or opening
some of those links escaped the product's production route.

## Cause

The browser router used `/wishlist` as its base, but server rendering used an
empty base. The server entry rewrites requests under `/wishlist` to local route
paths before SolidStart matches them, so simply setting a base for the server
router made its route matcher miss the page.

## Change

Pass the external, base-prefixed request URL to the server router while keeping
the existing internal request rewrite. The router can then render prefixed
links and match the route under the configured base path. The browser continues
to use its real URL and the same configured router base.

Candidate integration now checks that every same-origin internal link from the
board stays under the mounted path and exercises navigation to privacy/admin
and back.

## Validation

- `npm run typecheck`
- `WISHLIST_BASE_PATH=/wishlist npm run build`
- `make integration` — passed with the built API and web images at
  `http://127.0.0.1:18083/wishlist`; navigation and persistence phases passed.
- Built server HTML emits `/wishlist/privacy`, `/wishlist/admin`, and
  `/wishlist?view=...` for internal navigation.
