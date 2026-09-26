"""Only boundary between the pure Wishlist model and WNT MRL Runtime."""

from __future__ import annotations

from datetime import datetime, timedelta, timezone

from mrl_simulation_runtime.actors import Actor
from mrl_simulation_runtime.invariants import Invariant
from mrl_simulation_runtime.scenario import InitialScheduledAction, Scenario

from app.application.wishlist import Wishlist
from app.infrastructure.fakes import FakeClock, FakeOtpSender, MemoryEventStore, SequentialIds
from app.interfaces.admin_adapter import AdminAdapter, AdminRequest
from app.interfaces.visitor_adapter import VisitorAdapter, VisitorRequest


def create_simulation() -> Scenario:
    start = datetime(2026, 1, 1, 12, tzinfo=timezone.utc)
    store, clock, ids, sender = MemoryEventStore(), FakeClock(start), SequentialIds(), FakeOtpSender()
    wishlist = Wishlist(store, clock, ids, sender, admin_key="admin-simulation", otp_secret="simulation-secret")
    visitors = VisitorAdapter(wishlist)
    admins = AdminAdapter(wishlist)
    refs: dict[str, str] = {}
    observed = 0

    def action(name, actor, intention):
        def execute(context):
            nonlocal observed
            clock.set(context.clock.now())
            context.emit("actor_intention", name, source="wishlist-scenario", actor=actor,
                         correlation_id=name, payload={"intention": name})
            intention()
            for event in store.events[observed:]:
                # Private event payloads are deliberately not sent to runtime observations.
                context.emit("domain_event", event.name, source="Wishlist", actor=actor,
                             correlation_id=name, payload={"event_name": event.name})
            observed = len(store.events)
            context.emit("use_case_result", name, source="Wishlist", actor=actor,
                         correlation_id=name, payload={"succeeded": True})
        return execute

    def setup():
        created = admins.handle(AdminRequest("POST", "/v1/admin/apps", {"slug":"cat-care","name":"Cat Care"}, "admin-simulation"))
        assert created.status == 201
        refs["app"] = created.body["id"]
        for key,slug,title in (("a","family-sharing","Family sharing"),("b","export-health","Export health history")):
            response=admins.handle(AdminRequest("POST","/v1/admin/features",{"app_id":refs["app"],"slug":slug,"title":title},"admin-simulation"))
            assert response.status == 201
            refs[key]=response.body["id"]

    def verify_and_vote():
        visitors.handle(VisitorRequest("POST", "/v1/otp", body={"email": " Visitor@Example.com "}))
        verified = visitors.handle(VisitorRequest(
            "POST", "/v1/otp/verify",
            body={"email": "visitor@example.com", "code": sender.delivered["visitor@example.com"]}))
        refs["session"] = verified.establish_session or ""
        visitors.handle(VisitorRequest("POST", f"/v1/features/{refs['a']}/vote", session_id=refs["session"]))

    def suggest():
        submitted = visitors.handle(VisitorRequest("POST", "/v1/suggestions",
                                                    body={"app_id": refs["app"], "title": "Medication reminders"},
                                                    session_id=refs["session"]))
        refs["suggestion"] = submitted.body["suggestion_id"]
        assert len(wishlist.list_features("voting", "cat-care")) == 2

    def publish():
        response=admins.handle(AdminRequest("POST",f"/v1/admin/suggestions/{refs['suggestion']}/accept",
            {"slug":"medication-reminders","title":"Medication schedule","description":"Choose a time for each dose."},"admin-simulation"))
        assert response.status == 204
        edited = next(row for row in wishlist.list_features("voting", "cat-care")
                      if row["title"] == "Medication schedule")
        refs["new"] = edited["id"]
        assert edited["title"] == "Medication schedule"
        assert wishlist.list_features("voting", "cat-care")[0]["title"] == "Family sharing"
        assert wishlist.list_features("voting", "cat-care")[0]["id"] == refs["a"]

    def produce():
        # The admin can choose the lower-ranked feature.
        admins.handle(AdminRequest("PATCH",f"/v1/admin/features/{refs['b']}",
            {"status":"producing","title":"Export health history","description":""},"admin-simulation"))

    def deliver():
        admins.handle(AdminRequest("PATCH",f"/v1/admin/features/{refs['b']}",
            {"status":"delivered","title":"Export health history","description":"","delivery_url":"https://example.com/cat-care/release"},"admin-simulation"))
        assert wishlist.list_features("delivered", "cat-care")[0]["delivery_url"]

    steps = [
        ("setup", "admin", setup),
        ("verify-and-vote", "visitor", verify_and_vote),
        ("suggest-privately", "visitor", suggest),
        ("publish-suggestion", "admin", publish),
        ("select-lower-ranked", "admin", produce),
        ("mark-delivered", "admin", deliver),
    ]
    scheduled = [InitialScheduledAction(start + timedelta(minutes=i), action(name, actor, fn), name,
                                        source="wishlist-scenario", correlation_id=name)
                 for i, (name, actor, fn) in enumerate(steps)]

    def public_is_private(_context):
        rows = sum((wishlist.list_features(view) for view in ("voting", "producing", "delivered")), [])
        forbidden = {"email", "identity_id", "digest", "session_id", "suggestion_id"}
        return all(not (set(row) & forbidden) for row in rows)

    def unique_votes(_context):
        votes = wishlist.state.votes
        return len(votes) == len(set(votes))

    return Scenario(
        name="wishlist-demand-signal", seed=1, initial_time=start,
        run_id="wishlist-acceptance-001",
        actors=[Actor("visitor"), Actor("admin")],
        scheduled_actions=scheduled,
        invariants=[Invariant("public projection excludes private identity data", public_is_private),
                    Invariant("one active vote per identity and feature", unique_votes)],
    )
