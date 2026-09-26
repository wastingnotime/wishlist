# Domain background knowledge

The source handoff is preserved at `work/sources/wnt-wishlist-handoff.md`. It proposes a small public board with Voting, Producing, and Delivered views, app filtering, aggregate vote counts, verified email voting and suggestions, private moderation, and admin-controlled publication and lifecycle.

WNT's existing repository guidance says wishlist proposals are not product commitments. This simulation therefore models a demand signal and admin decisions without automatically promoting highly ranked features. Accepted implementation work belongs to an owning application project once a model release and repository decision exist.

Security-relevant background for later evaluation: public input needs bounds and safe rendering; OTP challenges need expiry and throttling; public responses must never include emails or OTP material; vote uniqueness must ultimately have a database constraint. The in-memory simulation can validate behavior, but it cannot prove browser, email-provider, or database properties.
