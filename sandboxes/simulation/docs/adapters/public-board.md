# Public board adapter contract

This contract is synchronized from the executable `app.interfaces.public_board` adapter. The adapter is technology-neutral and is not a production HTTP server.

## Requests

| Method | Path | Query | Result |
| --- | --- | --- | --- |
| `GET` | `/v1/apps` | — | `{ "apps": [...] }`; active apps only |
| `GET` | `/v1/features` | `view=voting\|producing\|delivered` (default `voting`), optional `app=<slug>` | `{ "features": [...] }`; app filter is identical for all views |

Features are ordered by active vote count descending, then publication time and id ascending. All results are public DTOs; identity, email, OTP, and suggestions are absent. Invalid view/filter returns `400 {"error":{"code":"invalid_request"}}`; unknown path returns `404`; unsupported method returns `405`.

## Browser state and freshness

The board URL is `/` with `view` and optional `app` query parameters. Missing `view` means `voting`. Selecting a view or app updates the URL so reload restores the selected filter. Each initial load and each filter change fetches the current board; refresh fetches current data while preserving URL state. Do not cache a prior vote ordering across a refresh.

The web project must validate these timing and refresh semantics in a real browser. This simulation contract does not claim browser behavior has already been verified.
