# Candidate slice map

| Slice | Domain concern | Current simulation status |
| --- | --- | --- |
| 01 | Apps and public board | Released in 0.1.0 with executable anonymous read adapter |
| 02 | Visitor OTP, sessions, vote toggle, private suggestion submission | Released in 0.2.0 with executable visitor adapter |
| 03 | Admin suggestion moderation and feature lifecycle | Domain behavior modeled; admin adapter contract remains to be built |
| 04 | Database/session/email production contracts | Deferred to later simulation and technology refinement |

The shared behavior is exercised in `sandboxes/simulation/docs/slices/01-demand-signal.md` and `02-visitor-writes.md`. Production persistence and security are explicitly outside the released simulation adapter contracts.
