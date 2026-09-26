package domain

import (
	"fmt"
	"regexp"
	"time"
)

type Status string

const (
	Voting    Status = "voting"
	Producing Status = "producing"
	Delivered Status = "delivered"
)

func (status Status) Valid() bool {
	return status == Voting || status == Producing || status == Delivered
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func ValidSlug(slug string) bool { return len(slug) <= 80 && slugPattern.MatchString(slug) }

type App struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Active      bool   `json:"-"`
}

type Feature struct {
	ID          string     `json:"id"`
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	AppID       string     `json:"-"`
	AppSlug     string     `json:"app_slug"`
	AppName     string     `json:"app_name"`
	Status      Status     `json:"status"`
	VoteCount   int        `json:"vote_count"`
	PublishedAt time.Time  `json:"published_at"`
	DeliveredAt *time.Time `json:"delivered_at"`
	DeliveryURL *string    `json:"delivery_url"`
}

type Board struct {
	apps     []App
	features []Feature
}

func NewBoard(apps []App, features []Feature) (Board, error) {
	knownApps := make(map[string]App, len(apps))
	for _, app := range apps {
		if app.ID == "" || !ValidSlug(app.Slug) || app.Name == "" {
			return Board{}, fmt.Errorf("invalid app seed")
		}
		if _, exists := knownApps[app.ID]; exists {
			return Board{}, fmt.Errorf("duplicate app id")
		}
		knownApps[app.ID] = app
	}
	for _, feature := range features {
		if feature.ID == "" || !ValidSlug(feature.Slug) || feature.Title == "" || !feature.Status.Valid() || feature.VoteCount < 0 {
			return Board{}, fmt.Errorf("invalid feature seed")
		}
		if _, exists := knownApps[feature.AppID]; !exists {
			return Board{}, fmt.Errorf("feature references unknown app")
		}
	}
	return Board{apps: append([]App(nil), apps...), features: append([]Feature(nil), features...)}, nil
}

func (board Board) Apps() []App {
	apps := make([]App, 0, len(board.apps))
	for _, app := range board.apps {
		if app.Active {
			apps = append(apps, app)
		}
	}
	return apps
}

func (board Board) Features(status Status, appSlug string) ([]Feature, error) {
	if !status.Valid() {
		return nil, fmt.Errorf("invalid feature status")
	}
	if appSlug != "" && !ValidSlug(appSlug) {
		return nil, fmt.Errorf("invalid app slug")
	}
	appByID := make(map[string]App, len(board.apps))
	for _, app := range board.apps {
		appByID[app.ID] = app
	}
	rows := make([]Feature, 0)
	for _, feature := range board.features {
		app := appByID[feature.AppID]
		if feature.Status != status || (appSlug != "" && app.Slug != appSlug) {
			continue
		}
		feature.AppSlug, feature.AppName = app.Slug, app.Name
		if status != Delivered {
			feature.DeliveredAt = nil
			feature.DeliveryURL = nil
		}
		rows = append(rows, feature)
	}
	// Stable insertion sort keeps this small in-memory read model dependency-free.
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && featureBefore(rows[j], rows[j-1]); j-- {
			rows[j], rows[j-1] = rows[j-1], rows[j]
		}
	}
	return rows, nil
}

func featureBefore(left, right Feature) bool {
	if left.VoteCount != right.VoteCount {
		return left.VoteCount > right.VoteCount
	}
	if !left.PublishedAt.Equal(right.PublishedAt) {
		return left.PublishedAt.Before(right.PublishedAt)
	}
	return left.ID < right.ID
}
