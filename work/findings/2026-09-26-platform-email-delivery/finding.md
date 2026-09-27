# Finding: platform-managed email delivery is unavailable

## Issue Type

- blocker

## Context

Production preparation for the Wishlist web app needs an email OTP sender.

## Observed Behavior

The production Wishlist API intentionally refuses to start until a real OTP
email provider is configured. Inspection of infra-platform's production
Terraform, Swarm stacks, provisioning scripts, IAM and ECR surfaces found no
active Amazon SES or Mailgun delivery integration. AWS budget notification
email addresses are present, but those do not provide application email
delivery.

## Expected Behavior

WNT applications can request transactional email through a platform-provided
service with a documented onboarding, sender, credential and operational
contract.

## Impact

Wishlist production release cannot send sign-in codes until the platform email
service and its app integration contract are available.

## Suspected Source

`wastingnotime/infra-platform` owns production AWS infrastructure and shared
runtime services.

## Evidence

- `terraform/environments/production/` contains no SES or Mailgun resources.
- `terraform/environments/production/swarm-deploy/stacks/` and its provisioning
  scripts contain no application mail delivery configuration.
- Search results for email notification configuration are limited to AWS
  budget notification addresses.

## Suggested Direction

Choose and provide one platform-managed delivery option (AWS SES or Mailgun),
including the application-facing contract, sender/domain onboarding, secret
delivery and operational ownership. This finding does not prescribe the
provider or its internal implementation.

## Owning Repository

`wastingnotime/infra-platform` (likely owner; confirm in the target issue).

## Local Impact

Wishlist keeps production launch blocked until the service contract and
credentials are available, then implements its OTP sender adapter against that
contract.

## Discord Anchor

- message link or ID: not available from Wishlist; target issue notification runs in infra-platform
- Discord Channel: `#blockers` (`1502715067186024628`)
- create anchor on open: unavailable in Wishlist; issue notification workflow
  follow-up recorded in the campaign handoff
- close reply on close: unavailable in Wishlist; issue notification workflow
  follow-up recorded in the campaign handoff

## Linked Target Issues

- target repository issue: https://github.com/wastingnotime/infra-platform/issues/658
- link back to local issue: https://github.com/wastingnotime/wishlist/issues/1
