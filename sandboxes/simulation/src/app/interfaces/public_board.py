"""Technology-neutral HTTP-shaped contract for anonymous public board reads."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

from app.application.wishlist import Wishlist
from app.domain.model import DomainError


@dataclass(frozen=True)
class PublicRequest:
    method: str
    path: str
    query: dict[str, str] = field(default_factory=dict)


@dataclass(frozen=True)
class PublicResponse:
    status: int
    body: dict[str, Any]


class PublicBoardAdapter:
    """Maps public board reads to use cases; it owns no domain decisions."""

    def __init__(self, wishlist: Wishlist) -> None:
        self.wishlist = wishlist

    def handle(self, request: PublicRequest) -> PublicResponse:
        if request.method != "GET":
            return PublicResponse(405, {"error": {"code": "method_not_allowed"}})
        if request.path == "/v1/apps":
            return PublicResponse(200, {"apps": self.wishlist.list_apps()})
        if request.path == "/v1/features":
            try:
                features = self.wishlist.list_features(
                    view=request.query.get("view", "voting"),
                    app_slug=request.query.get("app"),
                )
            except DomainError:
                return PublicResponse(400, {"error": {"code": "invalid_request"}})
            return PublicResponse(200, {"features": features})
        return PublicResponse(404, {"error": {"code": "not_found"}})
