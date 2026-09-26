"""Repository-owned Wishlist state reconstructed from private domain events."""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime
from typing import Any


class DomainError(ValueError):
    pass


@dataclass(frozen=True)
class Event:
    name: str
    at: datetime
    data: dict[str, Any]


@dataclass
class State:
    apps: dict[str, dict[str, Any]] = field(default_factory=dict)
    features: dict[str, dict[str, Any]] = field(default_factory=dict)
    suggestions: dict[str, dict[str, Any]] = field(default_factory=dict)
    identities: dict[str, str] = field(default_factory=dict)  # normalized email -> id
    challenges: dict[str, dict[str, Any]] = field(default_factory=dict)  # email -> challenge
    sessions: dict[str, str] = field(default_factory=dict)  # session -> identity id
    votes: set[tuple[str, str]] = field(default_factory=set)
    otp_requests: dict[str, list[datetime]] = field(default_factory=dict)

    def apply(self, event: Event) -> None:
        d = event.data
        n = event.name
        if n == "AppCreated":
            self.apps[d["id"]] = dict(d)
        elif n == "AppUpdated":
            self.apps[d["id"]].update(d["changes"])
        elif n == "FeaturePublished":
            self.features[d["id"]] = dict(d)
        elif n == "FeatureUpdated":
            self.features[d["id"]].update(d["changes"])
        elif n == "FeatureStatusChanged":
            feature = self.features[d["id"]]
            feature["status"] = d["status"]
            feature[d["status"] + "_at"] = event.at
            if "delivery_url" in d:
                feature["delivery_url"] = d["delivery_url"]
        elif n == "OtpRequested":
            self.challenges[d["email"]] = dict(d)
            self.otp_requests.setdefault(d["email"], []).append(event.at)
        elif n == "OtpAttempted":
            self.challenges[d["email"]]["attempts"] += 1
        elif n == "OtpVerified":
            self.challenges.pop(d["email"], None)
            self.identities[d["email"]] = d["identity_id"]
            self.sessions[d["session_id"]] = d["identity_id"]
        elif n == "VoteCast":
            self.votes.add((d["identity_id"], d["feature_id"]))
        elif n == "VoteRemoved":
            self.votes.remove((d["identity_id"], d["feature_id"]))
        elif n == "SuggestionSubmitted":
            self.suggestions[d["id"]] = dict(d)
        elif n == "SuggestionReviewed":
            self.suggestions[d["id"]].update(
                status=d["status"], resulting_feature_id=d.get("resulting_feature_id"), reviewed_at=event.at
            )
        else:
            raise DomainError(f"Unknown event: {n}")


def replay(events: list[Event]) -> State:
    state = State()
    for event in events:
        state.apply(event)
    return state
