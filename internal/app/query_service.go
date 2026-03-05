package app

import (
	"context"

	"MarketNews/internal/models"
)

type QueryService struct {
	newsRepo  NewsRepository
	eventRepo EventRepository
}

func NewQueryService(newsRepo NewsRepository, eventRepo EventRepository) *QueryService {
	return &QueryService{
		newsRepo:  newsRepo,
		eventRepo: eventRepo,
	}
}

func (s *QueryService) LoadNews(ctx context.Context, q models.EventQuery) ([]models.EventWithNews, error) {
	return s.eventRepo.List(ctx, q)
}

func (s *QueryService) ListNews(ctx context.Context, q models.NewsQuery) ([]models.NewsRecord, error) {
	return s.newsRepo.List(ctx, q)
}

func (s *QueryService) ListEvents(ctx context.Context, q models.EventQuery) ([]models.EventWithNews, error) {
	return s.LoadNews(ctx, q)
}

func (s *QueryService) GetNewsByID(ctx context.Context, id string) (*models.NewsRecord, error) {
	return s.newsRepo.GetByID(ctx, id)
}

func (s *QueryService) GetEventByID(ctx context.Context, id int64) (*models.EventRecord, error) {
	return s.eventRepo.GetByID(ctx, id)
}
