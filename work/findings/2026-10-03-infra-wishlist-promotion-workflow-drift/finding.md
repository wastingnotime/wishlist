# Blocker: Wishlist promotion intake expects the retired candidate workflow

## Issue Type

- blocker

## Context

Discovered while deploying the Wishlist privacy release candidate.

## Blocked Activity

- what is blocked: production promotion of the merged Wishlist candidate
- likely target repository: `wastingnotime/infra-platform`
- target certainty: confirmed

## Observed Behavior

The Wishlist candidate publication completed successfully, but the infra-platform `Wishlist image promotion intake` rejected both the push-triggered publication and a successful `workflow_dispatch` publication as not being a successful Wishlist main-branch candidate publication.

## Expected Behavior

The intake should accept a successful main-branch candidate publication run that matches the current Wishlist candidate workflow and its handoff artifact.

## Impact

The merged candidate images cannot advance into the reviewed production promotion pull request, so production deployment is blocked.

## Suspected Source

The infra-platform intake validator has an outdated workflow identity.

## Evidence

- Wishlist publication run: https://github.com/wastingnotime/wishlist/actions/runs/37149403513
- Run metadata: repository `wastingnotime/wishlist`; path `.github/workflows/ci.yml`; branch `main`; event `workflow_dispatch`; conclusion `success`; head SHA `8c93c5bb0ba92d594cb02194366ce0a5895eaae8`.
- Infra intake run: https://github.com/wastingnotime/infra-platform/actions/runs/37149598191
- Infra validator at `terraform/environments/production/swarm-deploy/scripts/prepare-wishlist-promotion.py` requires `WORKFLOW = ".github/workflows/publish-candidate.yml"`.
- Infra contract `docs/contracts/deployment/production-promotion-authority.md` says intake validates the successful candidate run and handoff, then opens a draft promotion pull request.

## Suggested Direction

Align the intake's accepted workflow identity with the current Wishlist publication workflow and rerun the promotion intake.

## Owning Repository

`wastingnotime/infra-platform` owns the promotion validator and intake workflow.

## Local Impact

Keep the candidate published and do not represent it as deployed. Resume promotion after the infra-owned intake contract is corrected.

## Discord Anchor

- message link or ID: not posted
- Discord Channel: `#blockers` (`1502715067186024628`)
- create anchor on open: not posted; external messaging is restricted in this session
- close reply on close: pending

## Linked Target Issues

- target repository issue: pending
- link back to local issue: pending
