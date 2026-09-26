# WNT Wishlist

Public demand board for ideas and feature requests across Wasting No Time apps.

Visitors can browse proposed, producing, and delivered features by app. Votes make demand visible; WNT retains the decision of what to build and when. Votes do not promise a roadmap commitment.

The MRL domain model lives in [the simulation](sandboxes/simulation/README.md). The web app is in `apps/web/`, backed by the Go API and PostgreSQL store in `apps/api/`.

## Run locally

Install web dependencies once with `cd apps/web && npm install`. Docker is required for the local PostgreSQL container. From the repository root run `make local`, then open `http://127.0.0.1:5173`. Ctrl-C stops the API and web app; PostgreSQL and its data remain available for the next run.

If you used the former SQLite app, stop the old local app and run `make import-sqlite` before restarting. It copies missing apps, features, identities, votes, and suggestions into PostgreSQL, including outstanding OTP/session records. You can rerun it to catch rows added before the switch; existing PostgreSQL rows are preserved. The source `apps/api/wishlist.db` is left untouched. Run `make test` for API and browser checks against a disposable PostgreSQL container.

The public board supports Voting, Producing, and Delivered views with app filters. Visitors can vote and submit private suggestions after email OTP verification. Admins can manage apps and features, review suggestions, and control the feature lifecycle at `/admin`.

For local development, OTP codes are printed in the API terminal and `/admin` accepts the local token `local-development-admin-token`. This development adapter is not suitable for real users. Production startup is blocked until an email OTP provider is configured. See the app READMEs for setup and validation.
