# Slice 02: verified visitor actions

- **Pack:** WNT MRL Python event-sourced simulation.
- **Runtime:** shared Wishlist scenario plus executable visitor HTTP-shaped adapter.
- **Architecture:** commands call existing use cases; opaque session establishment is an adapter effect; suggestions remain private.
- **Use cases:** OTP request/verification, current visitor session, vote toggle, suggestion submission.
- **Rules:** OTP response does not disclose email existence; active vote is unique by identity and feature; no new vote after Voting; removing a prior vote remains allowed; visitor suggestions begin pending and are absent from public queries.
- **Ports:** fake OTP sender and controlled clock; technology email/session/persistence implementations are not simulation concerns.
- **Scenario:** visitor verifies, votes, removes the vote, and submits a suggestion; public board remains unchanged by pending suggestion.
- **Tests:** anonymous session denial, generic OTP verification behavior, vote toggle and count, session vote list, feature status guard, suggestion privacy.
- **Done:** adapter contract tests pass and no public response/observation leaks private identity material.
- **Out of scope:** admin routes, actual cookie jar timing, real email, durable DB, production rate limits and abuse defense.
