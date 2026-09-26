# Candidate slice map

| Slice | Domain concern | Current simulation status |
| --- | --- | --- |
| 01 | Apps and public board | Modeled |
| 02 | Admin feature lifecycle | Modeled in shared environment |
| 03 | Email OTP identity | Modeled with fake sender and clock |
| 04 | Voting and ranking | Modeled with uniqueness invariant |
| 05 | Suggestions and moderation | Modeled privately |
| 06 | Delivered links | Modeled in public projection |

The combined first runnable scenario is documented in `sandboxes/simulation/docs/slices/01-demand-signal.md`. Channel and persistence adapter contracts require later refinement and model release.
