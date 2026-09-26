package application

import (
	"context"

	"github.com/wastingnotime/wishlist/apps/api/internal/domain"
)

type BoardReader interface {
	ListApps(context.Context) ([]domain.App, error)
	ListFeatures(context.Context, domain.Status, string) ([]domain.Feature, error)
}

type Board struct{ reader BoardReader }

func NewBoard(reader BoardReader) *Board { return &Board{reader: reader} }
func (board *Board) Apps(ctx context.Context) ([]domain.App, error) {
	return board.reader.ListApps(ctx)
}
func (board *Board) Features(ctx context.Context, status domain.Status, appSlug string) ([]domain.Feature, error) {
	return board.reader.ListFeatures(ctx, status, appSlug)
}

func (board *Board) Ready(ctx context.Context) error {
	if checker, ok := board.reader.(interface{ Ping(context.Context) error }); ok {
		return checker.Ping(ctx)
	}
	return nil
}
