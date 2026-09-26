# Visitor identity and write adapter contract

This executable route contract is implemented by `app.interfaces.visitor_adapter.VisitorAdapter`. It extends the public board adapter for verified identity, voting, and feature suggestions. A real API must map `establish_session` to an opaque HttpOnly session cookie; it must never return a reusable session id in JSON.

## Routes

| Method | Route | Auth | Behavior |
| --- | --- | --- | --- |
| `POST` | `/v1/otp` | Anonymous | Accept `{ "email": "..." }`; for valid email syntax always return `202 {"requested":true}` to avoid account enumeration. Apply request throttling. |
| `POST` | `/v1/otp/verify` | Anonymous | Accept email and code; single-use code returns `200 {"verified":true}` plus an opaque session-cookie effect. Invalid/expired code returns generic `400 invalid_or_expired_code`. |
| `GET` | `/v1/session` | Anonymous or verified session | Return `{ "verified": true, "vote_feature_ids": [...] }` for the current identity, or `{ "verified": false, "vote_feature_ids": [] }` when no valid session exists. |
| `POST` | `/v1/features/{feature_id}/vote` | Verified session | Toggle one vote. A new vote requires `voting`; removing an existing vote remains allowed after transition. Return `{ "voted": bool, "vote_count": number }`. |
| `POST` | `/v1/suggestions` | Verified session | Accept app id, title, and optional description; create a private pending suggestion and return `201 { "submitted": true, "suggestion_id": "..." }`. |

Public board routes remain anonymous. No visitor route returns email, OTP digest, other identities' votes, or pending suggestion content. The unique `(identity_id, feature_id)` rule must be enforced in persistent storage under concurrent requests.

## Browser semantics

After code verification, the server establishes a reasonably long-lived secure session. The browser should retry the intended vote after successful verification; refresh should restore the session's vote state. A suggestion confirmation must not imply public publication. OTP delivery timing and real session-cookie persistence require actual runtime/browser validation.
