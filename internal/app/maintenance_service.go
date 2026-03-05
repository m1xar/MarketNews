package app

import "context"

type MaintenanceService struct {
	repo MaintenanceRepository
}

func NewMaintenanceService(repo MaintenanceRepository) *MaintenanceService {
	return &MaintenanceService{repo: repo}
}

func (s *MaintenanceService) TruncateAll(ctx context.Context) error {
	return s.repo.TruncateAll(ctx)
}
