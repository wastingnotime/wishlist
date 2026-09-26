"""Deterministic ports for the simulation; never production adapters."""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone

from app.domain.model import DomainError, Event, replay


@dataclass
class MemoryEventStore:
    events: list[Event] = field(default_factory=list)

    def append(self, event: Event) -> None:
        if event.name == "VoteCast":
            vote = (event.data["identity_id"], event.data["feature_id"])
            if vote in replay(self.events).votes:
                raise DomainError("Active vote already exists")
        self.events.append(event)


@dataclass
class FakeClock:
    current: datetime = datetime(2026, 1, 1, tzinfo=timezone.utc)

    def now(self) -> datetime:
        return self.current

    def set(self, moment: datetime) -> None:
        self.current = moment


@dataclass
class SequentialIds:
    next_value: int = 0

    def new(self, prefix: str) -> str:
        self.next_value += 1
        return f"{prefix}-{self.next_value:04d}"


@dataclass
class FakeOtpSender:
    delivered: dict[str, str] = field(default_factory=dict)

    def send(self, email: str, code: str) -> None:
        self.delivered[email] = code
