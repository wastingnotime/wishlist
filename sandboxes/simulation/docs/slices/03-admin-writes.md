# Slice 03 — Admin catalog and moderation

## Goal

Expose a small authenticated boundary for the app catalog, published feature lifecycle, and private visitor suggestion moderation.

## Selected pack and shape

Continue the existing event-sourced Python MRL simulation. Extend the same model and runtime scenario; do not create another model or select a production technology here.

## Contracts and invariants

- Admin authorization is distinct from visitor OTP sessions.
- Suggestions stay private until an explicit accept action.
- Suggestions can be accepted as a new Voting feature, rejected, or merged into an existing feature from the same app.
- Only admins create apps/features and change lifecycle state.
- Lifecycle advances Voting → Producing → Delivered. Demand ranking never promotes a feature.
- Historical votes remain attached to the feature.

## Executable coverage

`app.interfaces.admin_adapter.AdminAdapter` provides HTTP-shaped route behavior. Tests cover unauthorized access and private suggestion moderation. `mrl_runtime_scenario.py` exercises admin routes alongside anonymous public reads and verified visitor writes.

## Exclusions

No production admin identity, key rotation, real email sender, storage engine, framework, deployment, or IP-level abuse controls are selected by this slice.
