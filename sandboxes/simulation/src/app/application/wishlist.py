"""Use cases and explicit public projection for the Wishlist hypothesis."""

from __future__ import annotations

import hashlib
import hmac
import re
from datetime import timedelta
from typing import Any
from urllib.parse import urlsplit

from app.domain.model import DomainError, Event, State, replay
from app.infrastructure.fakes import FakeClock, FakeOtpSender, MemoryEventStore, SequentialIds

STATUSES = ("voting", "producing", "delivered")
SUGGESTION_STATUSES = ("pending", "accepted", "rejected", "merged")


def _bounded(value: str, name: str, limit: int, *, optional: bool = False) -> str:
    value = value.strip()
    if (not optional and not value) or len(value) > limit:
        raise DomainError(f"Invalid {name}")
    return value


def _slug(value: str) -> str:
    value = _bounded(value.lower(), "slug", 80)
    if not re.fullmatch(r"[a-z0-9]+(?:-[a-z0-9]+)*", value):
        raise DomainError("Invalid slug")
    return value


def _url(value: str) -> str:
    value = _bounded(value, "URL", 500, optional=True)
    if value:
        parsed = urlsplit(value)
        if parsed.scheme not in ("http", "https") or not parsed.netloc or parsed.username or parsed.password:
            raise DomainError("Invalid URL")
    return value


class Wishlist:
    def __init__(
        self,
        store: MemoryEventStore,
        clock: FakeClock,
        ids: SequentialIds,
        sender: FakeOtpSender,
        *,
        admin_key: str,
        otp_secret: str,
    ) -> None:
        self.store, self.clock, self.ids, self.sender = store, clock, ids, sender
        self._admin_key, self._otp_secret = admin_key, otp_secret

    @property
    def state(self) -> State:
        return replay(self.store.events)

    def _record(self, event_name: str, **data: Any) -> Event:
        event = Event(event_name, self.clock.now(), data)
        self.store.append(event)
        return event

    def _admin(self, key: str) -> None:
        if not hmac.compare_digest(key, self._admin_key):
            raise DomainError("Admin authorization required")

    def _identity(self, session: str) -> str:
        identity = self.state.sessions.get(session)
        if not identity:
            raise DomainError("Verified session required")
        return identity

    def _app(self, app_id: str, *, active: bool = False) -> dict[str, Any]:
        app = self.state.apps.get(app_id)
        if not app or (active and not app["active"]):
            raise DomainError("App unavailable")
        return app

    def _feature(self, feature_id: str) -> dict[str, Any]:
        feature = self.state.features.get(feature_id)
        if not feature:
            raise DomainError("Feature unavailable")
        return feature

    def admin_create_app(self, key: str, slug: str, name: str, description: str = "", url: str = "") -> str:
        self._admin(key)
        slug = _slug(slug)
        if any(app["slug"] == slug for app in self.state.apps.values()):
            raise DomainError("App slug already exists")
        app_id = self.ids.new("app")
        self._record("AppCreated", id=app_id, slug=slug, name=_bounded(name, "name", 120),
                     description=_bounded(description, "description", 1000, optional=True),
                     url=_url(url), active=True,
                     created_at=self.clock.now(), updated_at=self.clock.now())
        return app_id

    def admin_update_app(self, key: str, app_id: str, **changes: Any) -> None:
        self._admin(key)
        self._app(app_id)
        allowed = {"name", "description", "url", "active"}
        if not changes or set(changes) - allowed:
            raise DomainError("Invalid app update")
        for field, limit in (("name", 120), ("description", 1000), ("url", 500)):
            if field in changes:
                changes[field] = _url(changes[field]) if field == "url" else _bounded(
                    changes[field], field, limit, optional=field != "name")
        if "active" in changes and not isinstance(changes["active"], bool):
            raise DomainError("Invalid active value")
        changes["updated_at"] = self.clock.now()
        self._record("AppUpdated", id=app_id, changes=changes)

    def list_apps(self) -> list[dict[str, Any]]:
        return [dict(id=a["id"], slug=a["slug"], name=a["name"], description=a["description"], url=a["url"])
                for a in sorted(self.state.apps.values(), key=lambda a: a["name"]) if a["active"]]

    def admin_create_feature(self, key: str, app_id: str, slug: str, title: str,
                             description: str = "", status: str = "voting", delivery_url: str = "") -> str:
        self._admin(key)
        self._app(app_id, active=True)
        slug = _slug(slug)
        if status not in STATUSES:
            raise DomainError("Invalid feature status")
        if any(f["app_id"] == app_id and f["slug"] == slug for f in self.state.features.values()):
            raise DomainError("Feature slug already exists in app")
        feature_id = self.ids.new("feature")
        now = self.clock.now()
        self._record("FeaturePublished", id=feature_id, app_id=app_id, slug=slug,
                     title=_bounded(title, "title", 160),
                     description=_bounded(description, "description", 2000, optional=True),
                     status=status, published_at=now,
                     producing_at=now if status == "producing" else None,
                     delivered_at=now if status == "delivered" else None,
                     delivery_url=_url(delivery_url),
                     created_at=now, updated_at=now)
        return feature_id

    def admin_update_feature(self, key: str, feature_id: str, **changes: str) -> None:
        self._admin(key)
        self._feature(feature_id)
        limits = {"title": 160, "description": 2000, "delivery_url": 500}
        if not changes or set(changes) - limits:
            raise DomainError("Invalid feature update")
        for field, value in changes.items():
            changes[field] = _url(value) if field == "delivery_url" else _bounded(
                value, field, limits[field], optional=field != "title")
        changes["updated_at"] = self.clock.now()
        self._record("FeatureUpdated", id=feature_id, changes=changes)

    def admin_change_feature_status(self, key: str, feature_id: str, status: str,
                                    delivery_url: str | None = None) -> None:
        self._admin(key)
        feature = self._feature(feature_id)
        if status not in STATUSES or status == feature["status"]:
            raise DomainError("Invalid lifecycle transition")
        if STATUSES.index(status) != STATUSES.index(feature["status"]) + 1:
            raise DomainError("Feature lifecycle must advance by one stage")
        data: dict[str, Any] = dict(id=feature_id, status=status)
        if delivery_url is not None:
            data["delivery_url"] = _url(delivery_url)
        self._record("FeatureStatusChanged", **data)

    def list_features(self, view: str = "voting", app_slug: str | None = None) -> list[dict[str, Any]]:
        if view not in STATUSES:
            raise DomainError("Invalid view")
        state = self.state
        if app_slug is not None:
            app_slug = _slug(app_slug)
        rows = []
        for feature in state.features.values():
            app = state.apps[feature["app_id"]]
            if feature["status"] != view or (app_slug is not None and app["slug"] != app_slug):
                continue
            count = sum(feature_id == feature["id"] for _, feature_id in state.votes)
            rows.append(dict(id=feature["id"], slug=feature["slug"], title=feature["title"],
                             description=feature["description"], app_slug=app["slug"],
                             app_name=app["name"], status=view, vote_count=count,
                             published_at=feature["published_at"].isoformat(),
                             delivered_at=feature["delivered_at"].isoformat() if feature["delivered_at"] else None,
                             delivery_url=feature["delivery_url"] if view == "delivered" else None))
        return sorted(rows, key=lambda row: (-row["vote_count"], row["published_at"], row["id"]))

    @staticmethod
    def normalize_email(email: str) -> str:
        value = email.strip().lower()
        if len(value) > 254 or not re.fullmatch(r"[^\s@]+@[^\s@]+\.[^\s@]+", value):
            raise DomainError("Invalid email")
        return value

    def _digest(self, challenge_id: str, code: str) -> str:
        return hmac.new(self._otp_secret.encode(), f"{challenge_id}:{code}".encode(), hashlib.sha256).hexdigest()

    def request_otp(self, email: str) -> None:
        email = self.normalize_email(email)
        now = self.clock.now()
        recent = [at for at in self.state.otp_requests.get(email, []) if now - at < timedelta(hours=1)]
        if len(recent) >= 3:
            raise DomainError("Too many OTP requests")
        challenge_id = self.ids.new("challenge")
        # Deterministic fake code; a production adapter must use a secure random generator.
        code = str(int(self._digest(challenge_id, "fake")[:12], 16) % 1_000_000).zfill(6)
        self.sender.send(email, code)
        self._record("OtpRequested", email=email, id=challenge_id, digest=self._digest(challenge_id, code),
                     expires_at=now + timedelta(minutes=10), attempts=0)

    def verify_otp(self, email: str, code: str) -> str:
        email = self.normalize_email(email)
        challenge = self.state.challenges.get(email)
        if not challenge or self.clock.now() >= challenge["expires_at"] or challenge["attempts"] >= 5:
            raise DomainError("Invalid or expired OTP")
        if not hmac.compare_digest(challenge["digest"], self._digest(challenge["id"], code)):
            self._record("OtpAttempted", email=email)
            raise DomainError("Invalid or expired OTP")
        identity_id = self.state.identities.get(email) or self.ids.new("identity")
        session_id = self.ids.new("session")
        self._record("OtpVerified", email=email, identity_id=identity_id, session_id=session_id)
        return session_id

    def vote_for_feature(self, session: str, feature_id: str) -> None:
        identity_id = self._identity(session)
        feature = self._feature(feature_id)
        if feature["status"] != "voting":
            raise DomainError("Feature is not accepting votes")
        if (identity_id, feature_id) in self.state.votes:
            raise DomainError("Vote already exists")
        self._record("VoteCast", identity_id=identity_id, feature_id=feature_id)

    def remove_vote(self, session: str, feature_id: str) -> None:
        identity_id = self._identity(session)
        if (identity_id, feature_id) not in self.state.votes:
            raise DomainError("Vote does not exist")
        self._record("VoteRemoved", identity_id=identity_id, feature_id=feature_id)

    def submit_suggestion(self, session: str, app_id: str, title: str, description: str = "") -> str:
        identity_id = self._identity(session)
        self._app(app_id, active=True)
        # Bounded per identity and simulated hour; provider-level throttling belongs to technology work.
        recent = sum(event.name == "SuggestionSubmitted" and event.data["identity_id"] == identity_id
                     and self.clock.now() - event.at < timedelta(hours=1) for event in self.store.events)
        if recent >= 3:
            raise DomainError("Too many suggestions")
        suggestion_id = self.ids.new("suggestion")
        self._record("SuggestionSubmitted", id=suggestion_id, app_id=app_id, identity_id=identity_id,
                     title=_bounded(title, "title", 160),
                     description=_bounded(description, "description", 2000, optional=True),
                     status="pending", created_at=self.clock.now(), reviewed_at=None,
                     resulting_feature_id=None)
        return suggestion_id

    def admin_list_suggestions(self, key: str, status: str = "pending") -> list[dict[str, Any]]:
        self._admin(key)
        if status not in SUGGESTION_STATUSES:
            raise DomainError("Invalid suggestion status")
        return [dict(suggestion) for suggestion in self.state.suggestions.values() if suggestion["status"] == status]

    def _pending(self, suggestion_id: str) -> dict[str, Any]:
        suggestion = self.state.suggestions.get(suggestion_id)
        if not suggestion or suggestion["status"] != "pending":
            raise DomainError("Pending suggestion required")
        return suggestion

    def admin_edit_suggestion(self, key: str, suggestion_id: str, *, title: str | None = None,
                              description: str | None = None) -> None:
        self._admin(key)
        self._pending(suggestion_id)
        changes: dict[str, str] = {}
        if title is not None:
            changes["title"] = _bounded(title, "title", 160)
        if description is not None:
            changes["description"] = _bounded(description, "description", 2000, optional=True)
        if not changes:
            raise DomainError("No suggestion changes provided")
        self._record("SuggestionEdited", id=suggestion_id, changes=changes)

    def admin_accept_suggestion(self, key: str, suggestion_id: str, slug: str,
                                title: str | None = None, description: str | None = None) -> str:
        self._admin(key)
        suggestion = self._pending(suggestion_id)
        if title is not None or description is not None:
            self.admin_edit_suggestion(key, suggestion_id, title=title, description=description)
            suggestion = self._pending(suggestion_id)
        feature_id = self.admin_create_feature(key, suggestion["app_id"], slug,
                                               suggestion["title"], suggestion["description"])
        self._record("SuggestionReviewed", id=suggestion_id, status="accepted", resulting_feature_id=feature_id)
        return feature_id

    def admin_reject_suggestion(self, key: str, suggestion_id: str) -> None:
        self._admin(key)
        self._pending(suggestion_id)
        self._record("SuggestionReviewed", id=suggestion_id, status="rejected")

    def admin_merge_suggestion(self, key: str, suggestion_id: str, feature_id: str) -> None:
        self._admin(key)
        suggestion = self._pending(suggestion_id)
        feature = self._feature(feature_id)
        if suggestion["app_id"] != feature["app_id"]:
            raise DomainError("Cannot merge across apps")
        self._record("SuggestionReviewed", id=suggestion_id, status="merged", resulting_feature_id=feature_id)
