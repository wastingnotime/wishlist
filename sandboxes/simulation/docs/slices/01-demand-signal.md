# Slice 01: demand signal and moderation

- **Pack:** WNT `mrl-simulation-project`, Python event-sourced simulation.
- **Runtime:** deterministic in-process use cases plus WNT MRL Runtime adapter.
- **Architecture:** one append-only event stream, reconstructed domain state, explicit public projection, fake OTP delivery and clock.
- **Discovery:** source handoff sections 1–16 and 21–24; repository proposal boundary.
- **Use cases:** admin app/feature management, public listing, OTP request/verification, vote/remove vote, submit/edit/review suggestion, feature lifecycle.
- **Rules:** private suggestions stay private; verified identities vote once; votes survive lifecycle; only admin publishes or moves features; public projections exclude identity material; vote ranking never changes lifecycle.
- **Ports:** event store, clock, ID generator, OTP sender, OTP digest secret. Real database and delivery provider are deferred.
- **Scenario:** handoff acceptance path from Cat Care creation through delivered link, including private suggestion and an admin choosing a lower-ranked feature.
- **Tests:** transition and permission errors, OTP expiry/single use, uniqueness, all-view app filtering, moderation privacy, ordering, complete acceptance path.
- **Done:** deterministic tests and runtime scenario pass; semantic observations show actor actions, domain events, and invariant results.
- **Out of scope:** web UI, HTTP contract, production auth, migrations, real email, deployment, model EGD/release.
