"""Only boundary between the pure Wishlist model and WNT MRL Runtime."""

from __future__ import annotations

from datetime import datetime, timedelta, timezone

from mrl_simulation_runtime.actors import Actor
from mrl_simulation_runtime.invariants import Invariant
from mrl_simulation_runtime.scenario import InitialScheduledAction, Scenario

from app.application.wishlist import Wishlist
from app.infrastructure.fakes import FakeClock, FakeOtpSender, MemoryEventStore, SequentialIds


def create_simulation() -> Scenario:
    start = datetime(2026, 1, 1, 12, tzinfo=timezone.utc)
    store, clock, ids, sender = MemoryEventStore(), FakeClock(start), SequentialIds(), FakeOtpSender()
    wishlist = Wishlist(store, clock, ids, sender, admin_key="admin-simulation", otp_secret="simulation-secret")
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
        refs["app"] = wishlist.admin_create_app("admin-simulation", "cat-care", "Cat Care")
        refs["a"] = wishlist.admin_create_feature("admin-simulation", refs["app"], "family-sharing", "Family sharing")
        refs["b"] = wishlist.admin_create_feature("admin-simulation", refs["app"], "export-health", "Export health history")

    def verify_and_vote():
        wishlist.request_otp(" Visitor@Example.com ")
        refs["session"] = wishlist.verify_otp("visitor@example.com", sender.delivered["visitor@example.com"])
        wishlist.vote_for_feature(refs["session"], refs["a"])

    def suggest():
        refs["suggestion"] = wishlist.submit_suggestion(refs["session"], refs["app"], "Medication reminders")
        assert len(wishlist.list_features("voting", "cat-care")) == 2

    def publish():
        refs["new"] = wishlist.admin_accept_suggestion("admin-simulation", refs["suggestion"], "medication-reminders")
        assert wishlist.list_features("voting", "cat-care")[0]["id"] == refs["a"]

    def produce():
        # The admin can choose the lower-ranked feature.
        wishlist.admin_change_feature_status("admin-simulation", refs["b"], "producing")

    def deliver():
        wishlist.admin_change_feature_status("admin-simulation", refs["b"], "delivered",
                                             delivery_url="https://example.com/cat-care/release")
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
