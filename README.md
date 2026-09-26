# WNT Wishlist

Public demand board for ideas and feature requests across Wasting No Time apps.

Visitors can browse proposed, producing, and delivered features by app. Votes make demand visible; WNT retains the decision of what to build and when. Votes do not promise a roadmap commitment.

The MRL domain model lives in [the simulation](sandboxes/simulation/README.md). The first implementation slice is the public board in `apps/web/`, backed by the authoritative API in `apps/api/`.

## Run locally

Start the API with `cd apps/api && go run ./cmd/api`, then start the web app with `cd apps/web && npm install && npm run dev -- --port 5173`. Open `http://127.0.0.1:5173`.

The current board uses deterministic sample data. It does not yet accept votes or suggestions.
