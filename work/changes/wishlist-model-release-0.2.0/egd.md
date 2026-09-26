# Wishlist 0.2.0 model expectation-gap detection

Date: 2026-09-26

## Compared evidence

- Wishlist 0.1.0 release and original product handoff.
- Visitor identity/write adapter contract and visitor slice contract.
- Existing use cases, projections, and OTP/vote/moderation tests.
- Nine passing deterministic simulation tests.
- 34-observation runtime scenario exercising visitor routes through verification, voting, private suggestion, admin edit/accept, and delivery.

## Coherence findings

1. Valid email OTP requests return the same accepted response regardless of whether an identity already exists; OTP verification failures use one generic error.
2. Successful verification establishes an opaque session effect. Session reads expose only the current identity's vote ids, while public feature reads remain anonymous.
3. Voting is a toggle at the adapter boundary: a unique vote is created only for Voting features, while removal remains possible after transition.
4. Suggestions are created privately and remain absent from public views until an admin accepts them.
5. Runtime observations record event names and actor intentions without private event payloads or email addresses.

## Gaps and dispositions

| Gap | Evidence | Disposition |
| --- | --- | --- |
| Session cookie attributes and persistence are not exercised by the simulation adapter. | Adapter models session establishment as an effect. | Web/API mapping requires opaque Secure, HttpOnly, SameSite cookie semantics and real browser validation. |
| OTP sender and rate controls are fakes/local rules; IP-level abuse behavior is absent. | In-memory deterministic provider and email-level request limits. | Production provider/throttling are technology work; retain generic responses and avoid logging secrets. |
| Duplicate-vote concurrency is not proven by a simulated sequential run. | In-memory store uniqueness guard. | Persistent API must enforce a unique database key on identity and feature. |
| Suggestions have no visitor-facing listing route. | Private moderation contract. | Intentional: no public query for pending/rejected suggestions. |

## EGD result

No gap contradicts the bounded visitor-write release intent. Recommend accepting model version 0.2.0 for deterministic identity, session, vote, and suggestion behavior. This release still does not prove production security, persistence, or browser cookie timing.
