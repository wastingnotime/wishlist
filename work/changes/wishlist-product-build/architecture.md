# Wishlist product build architecture

## Repository baseline

Before this change, the repository held only its wishlist entry README and repository guidance. The domain simulation now contains the accepted Wishlist 0.1.0 model and public board read adapter. No existing app code or reusable WNT UI package exists in this checkout.

## Reusable WNT conventions

- `cat-care` uses Go for the authoritative API and SolidStart/SolidJS for the browser app with a thin same-origin BFF.
- `contacts` uses the Go API service shape and keeps browser and API projects separate.
- `wnt-web` owns mature shared web primitives but currently exports no package to consume here. `wnt-design` owns technology-neutral foundations, not UI components.
- WNT product-shape guidance standardizes `apps/api`, `apps/web`, and later integration/runtime sandboxes.

## Decisions

- Build `apps/api` in Go with standard-library HTTP routing and an explicit application boundary.
- Build `apps/web` with SolidStart/SolidJS and a same-origin BFF, following the closest existing WNT app pattern.
- Begin with the public read slice: app list, three feature views, app filter, URL-preserved selection, and refresh from current API data.
- Use useful deterministic local sample data for this read-only first slice. It is not production persistence or a claim of live vote state. Durable storage, OTP, voting, suggestion moderation, and admin auth are subsequent slices.
- Defer `apps/mcp` for this human-facing MVP. No agent workflow or MCP-specific authorization/use case is in the handoff; revisit if agent-facing Wishlist operations become a requirement.
- Defer deployment, identity-provider selection, and persistence technology until the write slices establish their production constraints.

## First slice plan

1. Synchronize the released public board adapter into a Go API with `/healthz`, `/v1/apps`, and `/v1/features`.
2. Synchronize it into a responsive SolidStart browser board with Voting, Producing, Delivered, and app filtering.
3. Keep browser URLs as the source of selected view/app and fetch fresh API data on initial load, filter change, and refresh.
4. Verify service contracts, client behavior, and one real browser flow for filter/navigation/refresh.

This slice is intentionally read-only and does not claim the full handoff acceptance path is implemented.
