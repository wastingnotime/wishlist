# Repository Guidelines

This repository owns the WNT Wishlist web product and its domain simulation. Wishlist votes are a public demand signal; they do not commit WNT to build or schedule a feature.

## Repository Role

- Own Wishlist product behavior, application contracts, and implementation.
- Keep the MRL simulation as the behavioral source for synchronized application work.
- Keep API, browser, and any later MCP adapters within their project boundaries.
- Keep durable product decisions and change evidence in Markdown under `work/`.
- Keep shared WNT guidance in user-space capabilities and the WNT MCP surface.

## Scope Boundaries

- `sandboxes/simulation/` owns model extraction, refinement, validation, EGD, and model release.
- `apps/api/` owns authoritative HTTP behavior and persistence.
- `apps/web/` owns browser experience and a thin same-origin BFF.
- Public API responses must not expose visitor email, OTP, session, or moderation data.
- Do not copy WNT toolkit policy into the repository or depend on repository-local WNT installs.

## Working Rules

- Read `README.md` and the active change artifacts before extending product behavior.
- Use WNT capabilities for shared project shapes, security, and delivery guidance.
- Use conventional commit prefixes for committed changes.
- Keep the demand-signal principle explicit: ranking informs WNT judgment and never changes lifecycle state automatically.

## Working Rules

- Use `wnt capability show mcp-usage` when you need MCP guidance.
- Use `wnt capability show project-surfaces` when you need project-surface defaults.
- Use conventional commit prefixes for committed changes.
- Keep repository-specific rules local to this repo.

## Repository Binding

- This repository was created with `wnt create-repository`.
- WNT shared guidance lives outside the repository and is discovered from user-space.

## WNT Repository Defaults

Shared WNT guidance lives in user-space capabilities and the WNT MCP surface.

- Use `wnt capability show mcp-usage` for MCP guidance.
- Use `wnt capability show project-surfaces` for project-surface defaults.
- Use `wnt create-repository <name>` when creating a new repository.
- Keep WNT toolkit policy out of the repository-local source of truth.

<!-- wnt:wnt-toolkit-hints:start -->
## WNT Toolkit Hints

Shared WNT knowledge is not repository truth. Resolve current WNT guidance from
user or container space through WNT capabilities or the WNT MCP surface.

```bash
wnt capability list
wnt capability show <capability-name>
```

Start with these capabilities when the task is about WNT repository behavior:

```bash
wnt capability show ai-assisted-repository-baseline
wnt capability show project-surfaces
wnt capability show change-discipline
wnt capability show cross-repo-findings
wnt capability show issue-pr-linking
wnt capability show release-delivery-validation
```

This hint block is additive discovery guidance. It is not repository identity;
root strategic docs and the rest of `AGENTS.md` remain owned by the consuming
repository.
<!-- wnt:wnt-toolkit-hints:end -->
