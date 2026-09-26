"""Executable HTTP-shaped contract for visitor identity, votes, and suggestions."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any
from urllib.parse import unquote

from app.application.wishlist import Wishlist
from app.domain.model import DomainError


@dataclass(frozen=True)
class VisitorRequest:
    method: str
    path: str
    body: dict[str, Any] = field(default_factory=dict)
    session_id: str | None = None


@dataclass(frozen=True)
class VisitorResponse:
    status: int
    body: dict[str, Any]
    establish_session: str | None = None


class VisitorAdapter:
    """Maps visitor routes to use cases; session establishment is a cookie effect."""

    def __init__(self, wishlist: Wishlist) -> None:
        self.wishlist = wishlist

    def handle(self, request: VisitorRequest) -> VisitorResponse:
        if request.method == "GET" and request.path == "/v1/session":
            if not request.session_id:
                return VisitorResponse(401, {"error": {"code": "unauthenticated"}})
            identity_id = self.wishlist.state.sessions.get(request.session_id)
            if not identity_id:
                return VisitorResponse(401, {"error": {"code": "unauthenticated"}})
            votes = sorted(feature_id for voter_id, feature_id in self.wishlist.state.votes if voter_id == identity_id)
            return VisitorResponse(200, {"verified": True, "vote_feature_ids": votes})

        if request.method == "POST" and request.path == "/v1/otp":
            email = request.body.get("email", "")
            try:
                self.wishlist.request_otp(str(email))
            except DomainError:
                # For valid input, rate limiting and address existence do not alter the response.
                try:
                    self.wishlist.normalize_email(str(email))
                except DomainError:
                    return VisitorResponse(400, {"error": {"code": "invalid_request"}})
            return VisitorResponse(202, {"requested": True})

        if request.method == "POST" and request.path == "/v1/otp/verify":
            try:
                session = self.wishlist.verify_otp(str(request.body.get("email", "")),
                                                   str(request.body.get("code", "")))
            except DomainError:
                return VisitorResponse(400, {"error": {"code": "invalid_or_expired_code"}})
            return VisitorResponse(200, {"verified": True}, establish_session=session)

        if request.path.startswith("/v1/features/") and request.path.endswith("/vote"):
            if request.method != "POST":
                return VisitorResponse(405, {"error": {"code": "method_not_allowed"}})
            if not request.session_id:
                return VisitorResponse(401, {"error": {"code": "unauthenticated"}})
            feature_id = unquote(request.path.removeprefix("/v1/features/").removesuffix("/vote").strip("/"))
            if not feature_id or "/" in feature_id:
                return VisitorResponse(404, {"error": {"code": "not_found"}})
            try:
                identity_id = self.wishlist.state.sessions[request.session_id]
                vote = (identity_id, feature_id)
                if vote in self.wishlist.state.votes:
                    self.wishlist.remove_vote(request.session_id, feature_id)
                    active = False
                else:
                    self.wishlist.vote_for_feature(request.session_id, feature_id)
                    active = True
            except (DomainError, KeyError):
                return VisitorResponse(409, {"error": {"code": "vote_unavailable"}})
            count = sum(feature == feature_id for _, feature in self.wishlist.state.votes)
            return VisitorResponse(200, {"voted": active, "vote_count": count})

        if request.method == "POST" and request.path == "/v1/suggestions":
            if not request.session_id:
                return VisitorResponse(401, {"error": {"code": "unauthenticated"}})
            try:
                suggestion_id = self.wishlist.submit_suggestion(
                    request.session_id, str(request.body.get("app_id", "")),
                    str(request.body.get("title", "")), str(request.body.get("description", "")))
            except DomainError:
                return VisitorResponse(400, {"error": {"code": "invalid_request"}})
            return VisitorResponse(201, {"submitted": True, "suggestion_id": suggestion_id})

        return VisitorResponse(404, {"error": {"code": "not_found"}})
