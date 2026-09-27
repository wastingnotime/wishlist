# Admin login refinement check

The released simulation already models a separately authenticated admin and rejects visitor authority for catalog or moderation writes. Its adapter uses a bearer credential without assigning a production identity provider. Casdoor OIDC fills that adapter boundary: verified issuer, signature, audience, expiry, and configured subject determine admin authority. Browser login state never grants authority by itself.

No Wishlist domain event, public projection, vote rule, or moderation transition changes. The existing admin scenario remains the relevant model acceptance case, so this change does not require a new model release.

Validation: all ten simulation tests passed. Application checks cover signed-token verification, subject denial, expired/wrong-audience rejection, state and PKCE, encrypted browser session, local token isolation, and the existing admin browser flow against PostgreSQL.
