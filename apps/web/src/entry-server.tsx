import { identityPath } from "./components/identity";
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
          <link rel="icon" type="image/svg+xml" href={identityPath("favicon.svg")} />
          <link rel="icon" type="image/png" sizes="32x32" href={identityPath("favicon-32.png")} />
          <link rel="shortcut icon" href={identityPath("favicon.ico")} />
          <link rel="apple-touch-icon" sizes="180x180" href={identityPath("apple-touch-icon.png")} />
          <meta property="og:type" content="website" />
          <meta property="og:site_name" content="WNT Wishlist" />
          <meta property="og:title" content="Wishlist — Ideas are proposals. Votes are not promises." />
          <meta property="og:description" content="A public demand board for Wasting No Time app ideas and feature requests." />
          <meta property="og:image" content={`https://wastingnotime.org${identityPath("wishlist-social.png")}`} />
          <meta property="og:image:width" content="1200" />
          <meta property="og:image:height" content="630" />
          <meta property="og:image:alt" content="WNT Wishlist — Public demand board. Votes are not promises." />
          <meta name="twitter:card" content="summary_large_image" />
          <meta name="twitter:title" content="WNT Wishlist — Public demand board" />
          <meta name="twitter:image" content={`https://wastingnotime.org${identityPath("wishlist-social.png")}`} />
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
