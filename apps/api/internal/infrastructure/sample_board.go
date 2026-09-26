package infrastructure

import (
	"time"

	"github.com/wastingnotime/wishlist/apps/api/internal/domain"
)

func SampleBoard() (domain.Board, error) {
	catCare := domain.App{ID: "app-cat-care", Slug: "cat-care", Name: "Cat Care", Description: "Care tools for cats.", URL: "https://wastingnotime.org/apps/cat-care", Active: true}
	slidingTasks := domain.App{ID: "app-sliding-tasks", Slug: "sliding-tasks", Name: "Sliding Tasks", Description: "A calmer way to manage recurring work.", URL: "https://wastingnotime.org/apps/sliding-tasks", Active: true}
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	deliveredAt := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	deliveryURL := "https://wastingnotime.org/releases/cat-care/family-sharing"
	slidingDeliveryURL := "https://wastingnotime.org/releases/sliding-tasks/quick-add"
	features := []domain.Feature{
		{ID: "feature-family-sharing", AppID: catCare.ID, Slug: "family-sharing", Title: "Family sharing", Description: "Coordinate care with the people who help.", Status: domain.Voting, VoteCount: 184, PublishedAt: base.Add(-48 * time.Hour)},
		{ID: "feature-export-health", AppID: catCare.ID, Slug: "export-health-history", Title: "Export health history", Description: "Take a useful summary to an appointment.", Status: domain.Voting, VoteCount: 137, PublishedAt: base.Add(-24 * time.Hour)},
		{ID: "feature-medication-reminders", AppID: catCare.ID, Slug: "medication-reminders", Title: "Medication reminders", Description: "Keep a dependable schedule for each dose.", Status: domain.Voting, VoteCount: 82, PublishedAt: base.Add(-12 * time.Hour)},
		{ID: "feature-dark-mode", AppID: slidingTasks.ID, Slug: "dark-mode", Title: "Dark mode", Description: "A low-glare theme for late-night planning.", Status: domain.Voting, VoteCount: 41, PublishedAt: base.Add(-6 * time.Hour)},
		{ID: "feature-shared-calendar", AppID: catCare.ID, Slug: "shared-calendar", Title: "Shared care calendar", Description: "A clear view of upcoming care together.", Status: domain.Producing, VoteCount: 67, PublishedAt: base.Add(-120 * time.Hour)},
		{ID: "feature-batch-edit", AppID: slidingTasks.ID, Slug: "batch-edit", Title: "Batch editing", Description: "Move or update several tasks in one pass.", Status: domain.Producing, VoteCount: 38, PublishedAt: base.Add(-96 * time.Hour)},
		{ID: "feature-family-sharing-shipped", AppID: catCare.ID, Slug: "family-sharing-shipped", Title: "Family sharing", Description: "Share care responsibilities with your household.", Status: domain.Delivered, VoteCount: 98, PublishedAt: base.Add(-720 * time.Hour), DeliveredAt: &deliveredAt, DeliveryURL: &deliveryURL},
		{ID: "feature-quick-add-shipped", AppID: slidingTasks.ID, Slug: "quick-add-shipped", Title: "Quick add", Description: "Capture a task without leaving the current view.", Status: domain.Delivered, VoteCount: 54, PublishedAt: base.Add(-800 * time.Hour), DeliveredAt: &deliveredAt, DeliveryURL: &slidingDeliveryURL},
	}
	return domain.NewBoard([]domain.App{catCare, slidingTasks}, features)
}
