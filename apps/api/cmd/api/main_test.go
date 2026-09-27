package main

import "testing"

func TestStartupSeedIsEmptyInProduction(t *testing.T) {
	board, err := startupSeed(true)
	if err != nil {
		t.Fatal(err)
	}
	if apps := board.Apps(); len(apps) != 0 {
		t.Fatalf("production startup seeded %d sample apps", len(apps))
	}
	if features, err := board.Features("voting", ""); err != nil || len(features) != 0 {
		t.Fatalf("production startup seeded sample features: rows=%d err=%v", len(features), err)
	}
}

func TestStartupSeedIncludesLocalDemoCatalog(t *testing.T) {
	board, err := startupSeed(false)
	if err != nil {
		t.Fatal(err)
	}
	if apps := board.Apps(); len(apps) == 0 {
		t.Fatal("local startup seed has no demo apps")
	}
}
