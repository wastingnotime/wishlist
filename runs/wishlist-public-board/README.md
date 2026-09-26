# Public board slice validation

The anonymous board slice was validated against the local Go API and SolidStart app with Chromium. The browser flow covers Voting/Producing/Delivered, filters by app in all three views, restores query state after reload, follows a delivered link, refreshes current data, and checks an empty result.

Validation commands:

```bash
cd apps/api && go test ./...
cd apps/web && npm run typecheck && npm run build
cd apps/web && WNT_WEB_E2E_OUTPUT=../../runs/wishlist-public-board/browser-e2e.json WNT_WEB_E2E_ARTIFACT_DIR=../../runs/wishlist-public-board/artifacts npm run test:e2e
```

Result: API tests pass; browser typecheck and build pass; two Chromium flows pass with no browser console errors. The API uses deterministic in-memory sample data, so these results validate read behavior and browser composition only. They do not validate durable vote state or write operations.
