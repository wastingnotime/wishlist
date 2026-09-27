# Wishlist 0.4.0 model release decision

Date: 2026-09-26

**Accepted: Wishlist domain simulation model 0.4.0.** The simulation and its adapter contracts now reflect the materialized app's lifecycle, public visibility, visitor session, and moderation rules. Evidence: 13 passing deterministic tests, a completed MRL runtime scenario with both invariants passing, and `git diff --check`.

The model remains a proposal and demand signal. It does not turn wishlist entries into product commitments. Production persistence, Casdoor login, email delivery, and browser behavior are validated in their owning technology surfaces.
