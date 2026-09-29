# Visible app selection on the public board

The public board now shows one app at a time. A visible row of app links replaces the All apps dropdown, and the selected app name heads the board. On narrow screens the app row scrolls horizontally. The status links retain the selected app, and suggestion entry starts with that app selected.

When the URL has no app or names an inactive or missing app, the browser selects the first active app from the API's name-ordered list and replaces the URL with its app slug and current status view. This gives the homepage a deterministic, shareable scope without adding a repository-specific featured-app setting. The API's optional app filter remains available to other clients.

Validation: `npm run typecheck` and all four Playwright browser flows passed against a disposable PostgreSQL database. The browser flows cover the default URL, switching apps, status views, reload, an empty app, admin publishing, and visitor voting and suggestion submission.
