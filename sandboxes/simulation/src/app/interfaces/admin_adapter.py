"""Executable HTTP-shaped contract for authorized catalog and moderation writes."""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

from app.application.wishlist import Wishlist
from app.domain.model import DomainError


@dataclass(frozen=True)
class AdminRequest:
    method: str
    path: str
    body: dict[str, Any] = field(default_factory=dict)
    token: str = ""


@dataclass(frozen=True)
class AdminResponse:
    status: int
    body: dict[str, Any]


class AdminAdapter:
    """Maps admin HTTP-shaped requests to existing lifecycle use cases."""

    def __init__(self, wishlist: Wishlist) -> None:
        self.wishlist = wishlist

    def handle(self, request: AdminRequest) -> AdminResponse:
        try:
            self.wishlist._admin(request.token)
        except DomainError:
            return AdminResponse(401, {"error": {"code": "unauthorized"}})
        parts = request.path.strip("/").split("/")
        if request.method == "GET" and request.path == "/v1/admin/session":
            return AdminResponse(200, {"authorized": True})
        if request.method == "GET" and request.path == "/v1/admin/apps":
            apps = [dict(app) for app in sorted(self.wishlist.state.apps.values(), key=lambda item: item["name"])]
            return AdminResponse(200, {"apps": apps})
        if request.method == "GET" and request.path == "/v1/admin/suggestions":
            try:
                rows = self.wishlist.admin_list_suggestions(request.token)
            except DomainError:
                return AdminResponse(400, {"error": {"code": "invalid_request"}})
            fields = ("id", "app_id", "title", "description", "created_at")
            private_rows = [{field: row[field] for field in fields} for row in rows]
            return AdminResponse(200, {"suggestions": private_rows})
        try:
            if request.method == "POST" and parts == ["v1", "admin", "apps"]:
                app_id = self.wishlist.admin_create_app(request.token, str(request.body.get("slug", "")),
                    str(request.body.get("name", "")), str(request.body.get("description", "")), str(request.body.get("url", "")))
                return AdminResponse(201, {"id": app_id})
            if request.method == "PATCH" and len(parts) == 4 and parts[:3] == ["v1", "admin", "apps"]:
                self.wishlist.admin_update_app(request.token, parts[3], **request.body)
                return AdminResponse(204, {})
            if request.method == "POST" and parts == ["v1", "admin", "features"]:
                if set(request.body) - {"app_id", "slug", "title", "description"}:
                    return AdminResponse(400, {"error": {"code": "invalid_request"}})
                feature_id = self.wishlist.admin_create_feature(request.token, str(request.body.get("app_id", "")),
                    str(request.body.get("slug", "")), str(request.body.get("title", "")), str(request.body.get("description", "")))
                return AdminResponse(201, {"id": feature_id})
            if request.method == "PATCH" and len(parts) == 4 and parts[:3] == ["v1", "admin", "features"]:
                values = dict(request.body)
                status = values.pop("status", None)
                delivery_url = values.pop("delivery_url", None)
                current_status = self.wishlist.state.features[parts[3]]["status"]
                if status is not None and status != current_status:
                    self.wishlist.admin_change_feature_status(request.token, parts[3], str(status), delivery_url)
                elif delivery_url is not None:
                    values["delivery_url"] = delivery_url
                if values:
                    self.wishlist.admin_update_feature(request.token, parts[3], **values)
                return AdminResponse(204, {})
            if len(parts) == 5 and parts[:2] == ["v1", "admin"] and parts[2] == "suggestions" and request.method == "POST":
                suggestion_id, action = parts[3], parts[4]
                if action == "accept":
                    self.wishlist.admin_accept_suggestion(request.token, suggestion_id,
                        str(request.body.get("slug", "")), request.body.get("title"), request.body.get("description"))
                    return AdminResponse(204, {})
                if action == "reject":
                    self.wishlist.admin_reject_suggestion(request.token, suggestion_id)
                    return AdminResponse(204, {})
                if action == "merge":
                    self.wishlist.admin_merge_suggestion(request.token, suggestion_id, str(request.body.get("feature_id", "")))
                    return AdminResponse(204, {})
        except (DomainError, KeyError, TypeError):
            return AdminResponse(400, {"error": {"code": "invalid_request"}})
        return AdminResponse(404, {"error": {"code": "not_found"}})
