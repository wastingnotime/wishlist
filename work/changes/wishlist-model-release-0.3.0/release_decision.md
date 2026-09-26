# Wishlist 0.3.0 model release decision

Date: 2026-09-26

## Decision

**Accepted: Wishlist domain simulation model 0.3.0.** It adds the executable admin adapter while preserving the 0.1 public board and 0.2 visitor behavior. Evidence: 10 passing deterministic tests and the runtime acceptance scenario completed with all monitored invariants passing.

## Released behavior

- Admin-only app listing, creation, editing, and activation control.
- Admin-only feature creation, editing, and sequential Voting → Producing → Delivered transitions.
- Private suggestion listing, accept/edit/publish, reject, and same-app merge.
- Vote history remains attached through lifecycle transitions; vote rank does not change status.
- Anonymous session lookup returns a non-sensitive unverified state.

## Explicit non-claims

This release does not prove production email delivery, production admin authentication/key rotation, IP-level throttling, deployment readiness, or a production database. Those remain application and operations gates.

## Synchronization gate

API and web adapters may synchronize the released public, visitor, and admin contracts. They must provide their own HTTP security, persistent constraints, and real-browser evidence. Production rollout remains blocked until a real OTP email sender and production secrets are configured.
