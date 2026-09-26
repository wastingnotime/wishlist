from datetime import timedelta

import pytest

from app.application.wishlist import Wishlist
from app.domain.model import DomainError, Event
from app.infrastructure.fakes import FakeClock, FakeOtpSender, MemoryEventStore, SequentialIds


@pytest.fixture
def model():
    sender = FakeOtpSender()
    service = Wishlist(MemoryEventStore(), FakeClock(), SequentialIds(), sender,
                       admin_key="admin", otp_secret="test-secret")
    return service, sender


def verified(model, email="v@example.com"):
    service, sender = model
    service.request_otp(email)
    return service.verify_otp(email, sender.delivered[email])


def test_public_board_filter_and_ranking(model):
    service, _ = model
    cat = service.admin_create_app("admin", "cat-care", "Cat Care")
    tasks = service.admin_create_app("admin", "sliding-tasks", "Sliding Tasks")
    first = service.admin_create_feature("admin", cat, "first", "First")
    second = service.admin_create_feature("admin", cat, "second", "Second")
    service.admin_create_feature("admin", tasks, "other", "Other")
    cat_producing = service.admin_create_feature("admin", cat, "making", "Making", status="producing")
    service.admin_create_feature("admin", tasks, "making", "Making", status="producing")
    cat_delivered = service.admin_create_feature("admin", cat, "shipped", "Shipped", status="delivered")
    service.admin_create_feature("admin", tasks, "shipped", "Shipped", status="delivered")
    session = verified(model)
    service.vote_for_feature(session, second)
    assert [row["id"] for row in service.list_features("voting", "cat-care")] == [second, first]
    assert len(service.list_features("voting")) == 3
    for view in ("voting", "producing", "delivered"):
        assert all(row["app_slug"] == "cat-care" for row in service.list_features(view, "cat-care"))
    assert [row["id"] for row in service.list_features("producing", "cat-care")] == [cat_producing]
    assert [row["id"] for row in service.list_features("delivered", "cat-care")] == [cat_delivered]
    assert not any("email" in row or "identity_id" in row for row in service.list_features())


def test_otp_session_expiry_single_use_and_throttling(model):
    service, sender = model
    service.request_otp(" V@Example.com ")
    code = sender.delivered["v@example.com"]
    with pytest.raises(DomainError):
        service.verify_otp("v@example.com", "000000" if code != "000000" else "111111")
    session = service.verify_otp("v@example.com", code)
    assert service.state.sessions[session] == service.state.identities["v@example.com"]
    with pytest.raises(DomainError):
        service.verify_otp("v@example.com", code)
    service.request_otp("expired@example.com")
    service.clock.set(service.clock.now() + timedelta(minutes=10))
    with pytest.raises(DomainError):
        service.verify_otp("expired@example.com", sender.delivered["expired@example.com"])
    service.request_otp("limit@example.com")
    service.request_otp("limit@example.com")
    service.request_otp("limit@example.com")
    with pytest.raises(DomainError):
        service.request_otp("limit@example.com")


def test_vote_lifecycle_and_admin_selection(model):
    service, _ = model
    app = service.admin_create_app("admin", "cat-care", "Cat Care")
    popular = service.admin_create_feature("admin", app, "popular", "Popular")
    chosen = service.admin_create_feature("admin", app, "chosen", "Chosen")
    with pytest.raises(DomainError):
        service.admin_change_feature_status("admin", chosen, "delivered")
    session = verified(model)
    service.vote_for_feature(session, popular)
    service.vote_for_feature(session, chosen)
    with pytest.raises(DomainError):
        service.vote_for_feature(session, chosen)
    with pytest.raises(DomainError):
        service.store.append(Event("VoteCast", service.clock.now(),
                                   {"identity_id": service.state.sessions[session], "feature_id": chosen}))
    service.remove_vote(session, chosen)
    assert service.list_features()[1]["vote_count"] == 0
    service.vote_for_feature(session, chosen)
    with pytest.raises(DomainError):
        service.admin_change_feature_status("visitor", chosen, "producing")
    service.admin_change_feature_status("admin", chosen, "producing")
    with pytest.raises(DomainError):
        service.admin_change_feature_status("admin", chosen, "voting")
    assert not any(row["id"] == chosen for row in service.list_features("voting"))
    assert service.list_features("producing")[0]["vote_count"] == 1
    with pytest.raises(DomainError):
        service.vote_for_feature(session, chosen)
    service.admin_change_feature_status("admin", chosen, "delivered", delivery_url="https://example.com/release")
    delivered = service.list_features("delivered")[0]
    assert delivered["vote_count"] == 1
    assert delivered["delivery_url"] == "https://example.com/release"


def test_suggestions_are_private_until_admin_publishes(model):
    service, _ = model
    app = service.admin_create_app("admin", "cat-care", "Cat Care")
    session = verified(model)
    pending = service.submit_suggestion(session, app, "Medication reminders")
    assert service.list_features() == []
    with pytest.raises(DomainError):
        service.admin_list_suggestions("visitor")
    assert service.admin_list_suggestions("admin")[0]["id"] == pending
    feature = service.admin_accept_suggestion("admin", pending, "medication-reminders", title="Medication alerts")
    assert service.list_features()[0]["title"] == "Medication alerts"
    assert service.list_features()[0]["vote_count"] == 0
    assert service.admin_list_suggestions("admin", "accepted")[0]["resulting_feature_id"] == feature
    with pytest.raises(DomainError):
        service.admin_accept_suggestion("admin", pending, "again")
    with pytest.raises(DomainError):
        service.admin_edit_suggestion("admin", pending, title="Too late")
    rejected = service.submit_suggestion(session, app, "Reject me")
    service.admin_reject_suggestion("admin", rejected)
    assert len(service.list_features()) == 1
    merged = service.submit_suggestion(session, app, "Merge me")
    service.admin_merge_suggestion("admin", merged, feature)
    assert len(service.list_features()) == 1


def test_admin_can_edit_pending_suggestion_before_acceptance(model):
    service, _ = model
    app = service.admin_create_app("admin", "cat-care", "Cat Care")
    session = verified(model)
    suggestion = service.submit_suggestion(session, app, "Original", "Original details")
    service.admin_edit_suggestion("admin", suggestion, title="Reviewed title", description="Reviewed details")
    feature = service.admin_accept_suggestion("admin", suggestion, "reviewed-title")
    row = next(row for row in service.list_features() if row["id"] == feature)
    assert row["title"] == "Reviewed title"
    assert row["description"] == "Reviewed details"


def test_public_response_has_no_private_material(model):
    service, _ = model
    app = service.admin_create_app("admin", "cat-care", "Cat Care")
    feature = service.admin_create_feature("admin", app, "share", "Share")
    session = verified(model)
    service.vote_for_feature(session, feature)
    service.submit_suggestion(session, app, "Private idea")
    public = str(service.list_apps()) + str(service.list_features())
    assert "v@example.com" not in public
    assert "identity" not in public
    assert "digest" not in public
    assert "Private idea" not in public


def test_public_links_reject_unsafe_schemes(model):
    service, _ = model
    with pytest.raises(DomainError):
        service.admin_create_app("admin", "unsafe", "Unsafe", url="javascript:alert(1)")
    app = service.admin_create_app("admin", "safe", "Safe")
    feature = service.admin_create_feature("admin", app, "feature", "Feature")
    with pytest.raises(DomainError):
        service.admin_change_feature_status("admin", feature, "delivered", delivery_url="javascript:alert(1)")
