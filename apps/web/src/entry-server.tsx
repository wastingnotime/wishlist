import { createHandler, StartServer } from "@solidjs/start/server";

const basePath = import.meta.env.SERVER_BASE_URL.replace(/\/+$/, "");

export default createHandler(() => (
    <StartServer document={({ assets, children, scripts }) => (
      <html lang="en">
        <head>
          <meta charset="utf-8" />
          <meta name="viewport" content="width=device-width, initial-scale=1" />
          <meta name="theme-color" content="#0b0e11" />
          <meta name="description" content="A public demand board for Wasting No Time app ideas and feature requests." />
          <title>WNT Wishlist — Public demand board</title>
          {assets}
        </head>
        <body><div id="app">{children}</div>{scripts}</body>
      </html>
    )} />
  ), {}, async (event) => {
    if (!basePath) return;

    const requestURL = new URL(event.request.url);
    if (requestURL.pathname !== basePath && !requestURL.pathname.startsWith(`${basePath}/`)) return;

    requestURL.pathname = requestURL.pathname.slice(basePath.length) || "/";
    event.request = new Request(requestURL, event.request);
  });
