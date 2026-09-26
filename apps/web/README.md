# Wishlist web

SolidStart browser app with a same-origin BFF. The public board supports app filters, feature lifecycle views, vote actions, email OTP verification, and private suggestion submission. The `/admin` page supports app management, feature publishing and lifecycle, and pending suggestion review.

## Run locally

In one terminal, start the API:

```bash
cd apps/api
go run ./cmd/api
```

In another terminal, start the web app:

```bash
cd apps/web
npm install
npm run dev -- --port 5173
```

The API persists to `apps/api/wishlist.db` relative to its working directory. Local OTP codes appear in the API terminal. Visit `/admin` and use the local token `local-development-admin-token`. Configure `WISHLIST_API_URL` if the API is not at `http://127.0.0.1:8080`.

The API's OTP logger and local admin/OTP keys are for development only. Production API startup is blocked until a real email OTP sender is configured.

## Validate

```bash
npm run typecheck
npm run build
npm run test:e2e
```

The browser tests cover the public filters, lifecycle views, URL restoration, delivery links, and empty states. API tests exercise OTP, votes, privacy, admin moderation, and vote preservation through lifecycle transitions.
