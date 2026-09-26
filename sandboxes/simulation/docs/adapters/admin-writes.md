# Admin catalog and moderation adapter contract

`app.interfaces.admin_adapter.AdminAdapter` maps bearer-authorized admin requests to the existing Wishlist use cases. The transport token is separate from visitor OTP sessions. Unauthorized requests return `401`; admin data and mutation routes are never public.

| Method | Route | Contract |
| --- | --- | --- |
| GET | `/v1/admin/apps` | Return all apps, including inactive apps, for administration. |
| POST | `/v1/admin/apps` | Create `{slug,name,description?,url?}` and return `201 {id}`. |
| PATCH | `/v1/admin/apps/{id}` | Update name, description, URL, or active state; return `204`. |
| POST | `/v1/admin/features` | Publish a new Voting feature for an active app; return `201 {id}`. |
| PATCH | `/v1/admin/features/{id}` | Edit feature fields or advance exactly one lifecycle stage. A Delivered transition includes an HTTPS delivery URL. |
| GET | `/v1/admin/suggestions` | Return pending private suggestions without voter email or identity fields. |
| POST | `/v1/admin/suggestions/{id}/accept` | Optionally edit title/description and publish as a new Voting feature; return `204`. |
| POST | `/v1/admin/suggestions/{id}/reject` | Reject a pending suggestion; return `204`. |
| POST | `/v1/admin/suggestions/{id}/merge` | Mark a pending suggestion merged into an existing feature from the same app; accept `{feature_id}` and return `204`. |

Admin choice controls lifecycle. Vote rank never changes status. Moving a feature preserves its votes; lifecycle progression is Voting → Producing → Delivered. Public projections exclude all pending, rejected, accepted-history, and merged suggestion records.
