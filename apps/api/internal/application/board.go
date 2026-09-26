package application

import (
	"github.com/wastingnotime/wishlist/apps/api/internal/domain"
)

type BoardReader interface {
	Apps() []domain.App
	Features(domain.Status, string) ([]domain.Feature, error)
}

type Board struct{ reader BoardReader }

func NewBoard(reader BoardReader) *Board { return &Board{reader: reader} }
func (board *Board) Apps() []domain.App  { return board.reader.Apps() }
func (board *Board) Features(status domain.Status, appSlug string) ([]domain.Feature, error) {
	return board.reader.Features(status, appSlug)
}
