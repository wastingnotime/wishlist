# Unified Wishlist CI

Wishlist now follows the same three-stage CI shape as community-lab: source checks, built-image integration, and gated candidate publication. One `.github/workflows/ci.yml` replaces the separate quality and manual publish workflows. Pull requests run the first two jobs. A passing push to `main` publishes arm64 API and web images, records digest references, and dispatches the existing infra-platform promotion intake. Manual dispatch on `main` remains available for retries. Publication does not promote to production.

The repository-local `sandboxes/integration` stack builds the API and web images with disposable PostgreSQL. Chromium creates an app and feature through the built web UI at `/wishlist`, reloads the public board, then verifies the feature persists after API restart. It emits browser and validation evidence as CI artifacts. This development-mode scenario covers the candidate's public board and admin token path; production Casdoor and SMTP validation remain release gates.

The first built-image run exposed two production-path defects: server route matching needed the prefix removed while the server router still expected it, and server-rendered BFF requests used the external browser host from inside the web container. The server router now uses the stripped internal path, and production server-side BFF requests use the container's local web port. Browser routing retains the `/wishlist` base path.

Validation: `actionlint .github/workflows/ci.yml`, `make test`, and `make integration` passed locally. The integration result includes both browser phases with no console errors.
