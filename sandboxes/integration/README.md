# Wishlist candidate integration

This sandbox starts the built API and web images with disposable PostgreSQL. It runs the web image at the production `/wishlist` base path, then uses Chromium to create an app, publish a feature, reload the public board, restart the API, and confirm the feature remains visible.

From the repository root, install `apps/web` dependencies (`npm ci`) and Chromium (`npx playwright install chromium`), then run `make integration`. Browser evidence is written to `sandboxes/integration/artifacts/`. The stack uses a development-only admin token and OTP logger; production authentication and SMTP remain separate release gates. Production promotion is owned by infra-platform.
