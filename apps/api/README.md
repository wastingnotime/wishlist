# Wishlist API

Go API implementing the released public board read contract. HTTP handlers translate requests; the application layer calls the board read model; domain types define status and filter semantics.

## Run

```bash
go run ./cmd/api
```

The API listens on `127.0.0.1:8080`; set `WISHLIST_API_ADDR` to change the bind address.

## Routes

| Method | Route | Behavior |
| --- | --- | --- |
| `GET` | `/healthz` | liveness |
| `GET` | `/v1/apps` | active app list |
| `GET` | `/v1/features?view=voting&app=cat-care` | feature list; `view` defaults to `voting`, `app` is optional |

The three supported views are `voting`, `producing`, and `delivered`. Features are ordered by vote count descending, then publication time and id. Responses are `no-store` and never include identity or moderation data.

## Current slice

The service starts with deterministic sample apps and features in memory. Vote totals are sample values for exercising board ranking and display. Writes, authentication, durable storage, and live vote totals are later slices.

## Validate

```bash
go test ./...
```
