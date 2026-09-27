"""A causal Wishlist journey translated to the WNT MRL Runtime."""

from __future__ import annotations

from datetime import datetime, timedelta, timezone
from typing import Callable

from mrl_simulation_runtime.actors import Actor
from mrl_simulation_runtime.invariants import Invariant
from mrl_simulation_runtime.scenario import ObservatoryEdge, ObservatoryNode, Scenario

from app.application.wishlist import Wishlist
from app.infrastructure.fakes import FakeClock, FakeOtpSender, MemoryEventStore, SequentialIds
from app.interfaces.admin_adapter import AdminAdapter, AdminRequest
from app.interfaces.public_board import PublicBoardAdapter, PublicRequest
from app.interfaces.visitor_adapter import VisitorAdapter, VisitorRequest


START = datetime(2026, 1, 1, 12, tzinfo=timezone.utc)
ADMIN_KEY = "admin-simulation"


class Journey:
    """Actor behavior that schedules its first intention when the run starts."""

    def __init__(self, name: str, delay: timedelta, first: Callable) -> None:
        self.name, self.delay, self.first = name, delay, first

    def on_start(self, context) -> None:
        context.scheduler.schedule_after(
            context, self.delay, self.first, name=self.name, source=self.name,
            correlation_id=f"{self.name}-journey",
        )


def _observatory() -> tuple[list[ObservatoryNode], list[ObservatoryEdge]]:
    actors = ["curator", "maya", "leo", "noa"]
    use_cases = {
        "CreateApp": "catalog", "UpdateApp": "catalog", "PublishFeature": "catalog", "ViewBoard": "catalog",
        "RequestOTP": "identity", "VerifyOTP": "identity", "EndSession": "identity",
        "ToggleVote": "demand", "SubmitSuggestion": "demand",
        "ListSuggestions": "moderation", "ReviewSuggestion": "moderation", "AdvanceFeature": "lifecycle",
    }
    event_names = (
        "AppCreated", "AppUpdated", "FeaturePublished", "FeatureUpdated", "FeatureStatusChanged",
        "OtpRequested", "OtpVerified", "VoteCast", "VoteRemoved", "SuggestionSubmitted",
        "SuggestionReviewed", "SessionEnded",
    )
    # Numeric layer positions compensate for the runtime's current rank map:
    # its preset places inbound adapters before actors and merges outbound
    # adapters with projections. Keep this scenario's declared flow ordered.
    nodes = [ObservatoryNode(name, name.title(), "actor", -4, domain="wishlist", status="active")
             for name in actors]
    nodes += [
        ObservatoryNode("otp-provider", "Fake OTP Mail Provider", "outbound_adapter", 6.5, domain="wishlist"),
        ObservatoryNode("AdminAdapter", "Admin API Adapter", "inbound_adapter", -8, domain="wishlist"),
        ObservatoryNode("VisitorAdapter", "Visitor API Adapter", "inbound_adapter", -8, domain="wishlist"),
        ObservatoryNode("PublicBoardAdapter", "Public Board Adapter", "inbound_adapter", -8, domain="wishlist"),
    ]
    nodes += [ObservatoryNode(name, name, "use_case", "use_cases", domain="wishlist",
                              description=f"{group} use case")
              for name, group in use_cases.items()]
    nodes += [ObservatoryNode("WishlistDomain", "Wishlist Domain", "aggregate", "domain_model", domain="wishlist"),
              ObservatoryNode("EventStore", "Event Store", "repository", "event_store", domain="wishlist"),
              ObservatoryNode("PublicBoard", "Public Board", "projection", "projections", domain="wishlist"),
              ObservatoryNode("ObservationLog", "Observation Log", "projection", "projections", domain="wishlist")]
    nodes += [ObservatoryNode(name, name, "event", "events", domain="wishlist")
              for name in event_names]
    edges = [ObservatoryEdge("curator", "AdminAdapter", "admin commands"),
             ObservatoryEdge("maya", "VisitorAdapter", "visitor commands"),
             ObservatoryEdge("leo", "VisitorAdapter", "visitor commands"),
             ObservatoryEdge("noa", "VisitorAdapter", "visitor commands"),
             ObservatoryEdge("VisitorAdapter", "RequestOTP", "request code"),
             ObservatoryEdge("RequestOTP", "otp-provider", "send code"),
             ObservatoryEdge("PublicBoardAdapter", "PublicBoard", "read")]
    edges += [ObservatoryEdge("AdminAdapter" if name in {"CreateApp", "UpdateApp", "PublishFeature", "ListSuggestions", "ReviewSuggestion", "AdvanceFeature"}
                              else "PublicBoardAdapter" if name == "ViewBoard" else "VisitorAdapter",
                              name, "invoke") for name in use_cases]
    edges += [ObservatoryEdge(name, "PublicBoard" if name == "ViewBoard" else "WishlistDomain", "decision")
              for name in use_cases]
    edges += [ObservatoryEdge("WishlistDomain", name, "records") for name in event_names]
    edges += [ObservatoryEdge(name, "EventStore", "append") for name in event_names]
    edges += [ObservatoryEdge("WishlistDomain", "EventStore", "append events"),
              ObservatoryEdge("EventStore", "PublicBoard", "project"),
              ObservatoryEdge("PublicBoard", "ObservationLog", "observe")]
    return nodes, edges


def create_simulation() -> Scenario:
    store, clock, ids, sender = MemoryEventStore(), FakeClock(START), SequentialIds(), FakeOtpSender()
    wishlist = Wishlist(store, clock, ids, sender, admin_key=ADMIN_KEY, otp_secret="simulation-secret")
    visitors, admins, board = VisitorAdapter(wishlist), AdminAdapter(wishlist), PublicBoardAdapter(wishlist)
    refs: dict[str, str] = {}
    observed = 0

    def adapter_for(use_case: str) -> str:
        if use_case in {"CreateApp", "UpdateApp", "PublishFeature", "ListSuggestions", "ReviewSuggestion", "AdvanceFeature"}:
            return "AdminAdapter"
        return "PublicBoardAdapter" if use_case == "ViewBoard" else "VisitorAdapter"

    def later(context, delay: timedelta, actor: str, label: str, callback: Callable) -> None:
        context.scheduler.schedule_after(context, delay, callback, name=label, source=actor,
                                         correlation_id=f"{actor}-journey")

    def invoke(context, actor: str, use_case: str, intention: str, operation: Callable, expected: int):
        nonlocal observed
        clock.set(context.clock.now())
        correlation = f"{actor}-{intention}"
        context.emit("actor_intention", intention, source=actor, actor=actor,
                     correlation_id=correlation, payload={"use_case_id": use_case})
        context.emit("use_case_invoked", use_case, source=adapter_for(use_case), actor=actor,
                     correlation_id=correlation, payload={"use_case_id": use_case})
        response = operation()
        assert response.status == expected, f"{correlation}: expected {expected}, got {response.status}"
        for event in store.events[observed:]:
            # Event payloads can contain email, OTP digests, and session ids.
            context.emit("domain_event", event.name, source="WishlistDomain", actor=actor,
                         correlation_id=correlation, payload={"use_case": use_case})
        observed = len(store.events)
        context.emit("use_case_result", use_case, source=use_case, actor=actor,
                     correlation_id=correlation, payload={"status": "succeeded", "http_status": expected})
        return response

    def admin(context, use_case: str, intention: str, method: str, path: str,
              body: dict | None = None, expected: int = 204):
        return invoke(context, "curator", use_case, intention,
                      lambda: admins.handle(AdminRequest(method, path, body or {}, ADMIN_KEY)), expected)

    def visitor(context, actor: str, use_case: str, intention: str, method: str,
                path: str, body: dict | None = None, expected: int = 200):
        return invoke(context, actor, use_case, intention,
                      lambda: visitors.handle(VisitorRequest(method, path, body or {}, refs.get(f"{actor}_session"))), expected)

    def read_board(context, actor: str, view: str, expected_count: int):
        result = invoke(context, actor, "ViewBoard", f"view-{view}-board",
                        lambda: board.handle(PublicRequest("GET", "/v1/features", {"view": view, "app": "cat-care"})), 200)
        assert len(result.body["features"]) == expected_count
        return result.body["features"]

    def seed_catalog(context):
        created = admin(context, "CreateApp", "create-cat-care", "POST", "/v1/admin/apps",
                        {"slug": "cat-care", "name": "Cat Care"}, 201)
        refs["app"] = created.body["id"]
        for key, slug, title in (("popular", "family-sharing", "Family sharing"),
                                  ("lower", "export-health", "Export health history")):
            result = admin(context, "PublishFeature", f"publish-{slug}", "POST", "/v1/admin/features",
                           {"app_id": refs["app"], "slug": slug, "title": title}, 201)
            refs[key] = result.body["id"]
        read_board(context, "curator", "voting", 2)
        later(context, timedelta(minutes=8), "curator", "moderate-suggestions", moderate)

    def request_code(context, actor: str, email: str):
        visitor(context, actor, "RequestOTP", "request-code", "POST", "/v1/otp", {"email": email}, 202)
        assert email in sender.delivered
        context.emit("external_effect", "otp-provider", source="VisitorAdapter", actor="otp-provider",
                     correlation_id=f"{actor}-request-code", payload={"delivery": "accepted", "use_case_id": "RequestOTP"})
        later(context, timedelta(minutes=1), actor, "verify-email", lambda next_context: verify(next_context, actor, email))

    def verify(context, actor: str, email: str):
        result = visitor(context, actor, "VerifyOTP", "verify-email", "POST", "/v1/otp/verify",
                         {"email": email, "code": sender.delivered[email]}, 200)
        assert result.establish_session
        refs[f"{actor}_session"] = result.establish_session
        later(context, timedelta(minutes=1), actor, "vote", lambda next_context: vote(next_context, actor))

    def vote(context, actor: str):
        feature = refs["lower"] if actor == "maya" else refs["popular"]
        result = visitor(context, actor, "ToggleVote", "support-feature", "POST", f"/v1/features/{feature}/vote")
        assert result.body["voted"] is True
        later(context, timedelta(minutes=1), actor, "suggest", lambda next_context: suggest(next_context, actor))

    def suggest(context, actor: str):
        title = {"maya": "Medication reminders", "leo": "Family calendar", "noa": "Unrelated request"}[actor]
        result = visitor(context, actor, "SubmitSuggestion", "suggest-privately", "POST", "/v1/suggestions",
                         {"app_id": refs["app"], "title": title}, 201)
        refs[f"{actor}_suggestion"] = result.body["suggestion_id"]
        assert all(row["title"] != title for row in read_board(context, actor, "voting", 2))

    def moderate(context):
        queue = admin(context, "ListSuggestions", "inspect-pending-queue", "GET", "/v1/admin/suggestions", expected=200)
        assert {row["id"] for row in queue.body["suggestions"]} == {
            refs[f"{actor}_suggestion"] for actor in ("maya", "leo", "noa")}
        admin(context, "ReviewSuggestion", "accept-medication-reminders", "POST",
              f"/v1/admin/suggestions/{refs['maya_suggestion']}/accept",
              {"slug": "medication-reminders", "title": "Medication schedule",
               "description": "Choose a time for each dose."})
        admin(context, "ReviewSuggestion", "merge-family-calendar", "POST",
              f"/v1/admin/suggestions/{refs['leo_suggestion']}/merge", {"feature_id": refs["popular"]})
        admin(context, "ReviewSuggestion", "reject-unrelated-request", "POST",
              f"/v1/admin/suggestions/{refs['noa_suggestion']}/reject")
        rows = read_board(context, "curator", "voting", 3)
        assert rows[0]["id"] == refs["popular"] and rows[0]["vote_count"] == 2
        assert next(row for row in rows if row["title"] == "Medication schedule")["vote_count"] == 0
        later(context, timedelta(minutes=2), "curator", "choose-lower-ranked", produce)

    def produce(context):
        admin(context, "AdvanceFeature", "select-lower-ranked-feature", "PATCH",
              f"/v1/admin/features/{refs['lower']}",
              {"status": "producing", "title": "Export health history", "description": ""})
        assert read_board(context, "curator", "producing", 1)[0]["id"] == refs["lower"]
        later(context, timedelta(minutes=1), "curator", "deliver-feature", deliver)

    def deliver(context):
        admin(context, "AdvanceFeature", "mark-feature-delivered", "PATCH",
              f"/v1/admin/features/{refs['lower']}",
              {"status": "delivered", "title": "Export health history", "description": "",
               "delivery_url": "https://example.com/cat-care/release"})
        rows = read_board(context, "curator", "delivered", 1)
        assert rows[0]["delivery_url"] == "https://example.com/cat-care/release"
        later(context, timedelta(minutes=1), "maya", "remove-historical-vote", remove_vote)
        later(context, timedelta(minutes=2), "curator", "deactivate-app", deactivate)

    def remove_vote(context):
        result = visitor(context, "maya", "ToggleVote", "remove-historical-vote", "POST",
                         f"/v1/features/{refs['lower']}/vote")
        assert result.body == {"voted": False, "vote_count": 0}
        visitor(context, "maya", "ToggleVote", "new-vote-after-delivery-denied", "POST",
                f"/v1/features/{refs['lower']}/vote", expected=409)
        later(context, timedelta(minutes=3), "maya", "logout", logout)

    def deactivate(context):
        admin(context, "UpdateApp", "deactivate-cat-care", "PATCH", f"/v1/admin/apps/{refs['app']}",
              {"name": "Cat Care", "description": "", "url": "", "active": False})
        assert read_board(context, "curator", "delivered", 0) == []
        later(context, timedelta(minutes=1), "curator", "reactivate-app", reactivate)

    def reactivate(context):
        admin(context, "UpdateApp", "reactivate-cat-care", "PATCH", f"/v1/admin/apps/{refs['app']}",
              {"name": "Cat Care", "description": "", "url": "", "active": True})
        assert read_board(context, "curator", "delivered", 1)[0]["id"] == refs["lower"]

    def logout(context):
        visitor(context, "maya", "EndSession", "logout", "DELETE", "/v1/session", expected=204)
        result = visitor(context, "maya", "EndSession", "check-logged-out", "GET", "/v1/session")
        assert result.body["verified"] is False
        later(context, timedelta(days=30), "leo", "check-session-expiry", expired_session)

    def expired_session(context):
        result = visitor(context, "leo", "EndSession", "check-expired-session", "GET", "/v1/session")
        assert result.body["verified"] is False
        visitor(context, "leo", "ToggleVote", "expired-vote-denied", "POST",
                f"/v1/features/{refs['popular']}/vote", expected=401)

    def public_privacy(_context):
        rows = sum((wishlist.list_features(view) for view in ("voting", "producing", "delivered")), [])
        forbidden = {"email", "identity_id", "digest", "session_id", "suggestion_id"}
        return all(not (set(row) & forbidden) for row in rows)

    def valid_lifecycle(_context):
        state = wishlist.state
        return all(feature["status"] in {"voting", "producing", "delivered"}
                   and (feature["status"] != "delivered" or
                        (feature["producing_at"] is not None and feature["delivery_url"].startswith("https://")))
                   for feature in state.features.values())

    def vote_history_consistent(_context):
        active: set[tuple[str, str]] = set()
        for event in store.events:
            if event.name not in {"VoteCast", "VoteRemoved"}:
                continue
            vote = (event.data["identity_id"], event.data["feature_id"])
            if event.name == "VoteCast":
                if vote in active:
                    return False
                active.add(vote)
            else:
                if vote not in active:
                    return False
                active.remove(vote)
        return active == wishlist.state.votes

    nodes, edges = _observatory()
    return Scenario(
        name="wishlist-demand-and-decision-journey", seed=1, initial_time=START,
        run_id="wishlist-journey-002",
        actors=[
            Actor("curator", Journey("curator", timedelta(0), seed_catalog)),
            Actor("maya", Journey("maya", timedelta(minutes=1),
                                  lambda context: request_code(context, "maya", "maya@example.com"))),
            Actor("leo", Journey("leo", timedelta(minutes=2),
                                 lambda context: request_code(context, "leo", "leo@example.com"))),
            Actor("noa", Journey("noa", timedelta(minutes=2),
                                 lambda context: request_code(context, "noa", "noa@example.com"))),
            Actor("otp-provider"),
        ],
        invariants=[
            Invariant("public projection excludes private identity data", public_privacy),
            Invariant("vote history preserves one active vote per identity and feature", vote_history_consistent),
            Invariant("features follow valid lifecycle and delivery links", valid_lifecycle),
        ],
        observatory_nodes=nodes, observatory_edges=edges,
    )
