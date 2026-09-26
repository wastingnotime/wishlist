# WNT Wishlist

Public demand board for ideas and feature requests across Wasting No Time apps.

Visitors can browse proposed, producing, and delivered features by app. Votes make demand visible; WNT retains the decision of what to build and when. Votes do not promise a roadmap commitment.

The MRL domain model lives in [the simulation](sandboxes/simulation/README.md). The web app is in `apps/web/`, backed by the Go API and SQLite store in `apps/api/`.

## Run locally

Install web dependencies once with `cd apps/web && npm install`, then from the repository root run `make local`. This starts the API and web app together; stop both with Ctrl-C. Open `http://127.0.0.1:5173`.

The public board supports Voting, Producing, and Delivered views with app filters. Visitors can vote and submit private suggestions after email OTP verification. Admins can manage apps and features, review suggestions, and control the feature lifecycle at `/admin`.

For local development, OTP codes are printed in the API terminal and `/admin` accepts the local token `local-development-admin-token`. This development adapter is not suitable for real users. Production startup is blocked until an email OTP provider is configured. See the app READMEs for setup and validation.
