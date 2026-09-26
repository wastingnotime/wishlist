# Wishlist web

SolidStart browser app with a same-origin BFF. It owns the public board view, URL state, and browser refresh behavior. The API remains authoritative for feature and app data.

## Run locally

Start the API in one terminal:

```bash
cd apps/api
go run ./cmd/api
```

Start the web app in another terminal:

```bash
cd apps/web
npm install
npm run dev -- --port 5173
```

Set `WISHLIST_API_URL` if the API is not at `http://127.0.0.1:8080`.

## Current slice

The public board supports Voting, Producing, and Delivered views, filtering by app, URL-preserved selection, and refresh. The API currently starts with deterministic sample data in memory. Visitor verification, votes, suggestions, admin tools, and durable persistence are not implemented in this slice.

## Validation

```bash
npm run typecheck
npm run build
npm run test:e2e
```

The browser tests start both the API and web dev server and cover app filtering across views, URL restoration after reload, delivery links, and empty states.
