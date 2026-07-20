package services

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"nonsoemeka-backend/internal/dto"
	"nonsoemeka-backend/internal/repository"
)

type ReportsService interface {
	GetSalesTrends(ctx context.Context, startDate, endDate string, page, pageSize int) (dto.PaginatedResponse[dto.SalesTrendItem], error)
	GetTopProducts(ctx context.Context, startDate, endDate string, page, pageSize int) (dto.PaginatedResponse[dto.TopProductItem], error)
}

type reportsService struct {
	pool     *pgxpool.Pool
	saleRepo repository.SaleRepository
}

func NewReportsService(pool *pgxpool.Pool, saleRepo repository.SaleRepository) ReportsService {
	return &reportsService{
		pool:     pool,
		saleRepo: saleRepo,
	}
}

func (s *reportsService) GetSalesTrends(ctx context.Context, startDate, endDate string, page, pageSize int) (dto.PaginatedResponse[dto.SalesTrendItem], error) {
	trends, total, err := s.saleRepo.GetSalesTrends(ctx, s.pool, startDate, endDate, page, pageSize)
	if err != nil {
		return dto.PaginatedResponse[dto.SalesTrendItem]{}, err
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	return dto.PaginatedResponse[dto.SalesTrendItem]{
		Data: trends,
		Pagination: dto.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *reportsService) GetTopProducts(ctx context.Context, startDate, endDate string, page, pageSize int) (dto.PaginatedResponse[dto.TopProductItem], error) {
	topProducts, total, err := s.saleRepo.GetTopProducts(ctx, s.pool, startDate, endDate, page, pageSize)
	if err != nil {
		return dto.PaginatedResponse[dto.TopProductItem]{}, err
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}

	return dto.PaginatedResponse[dto.TopProductItem]{
		Data: topProducts,
		Pagination: dto.PaginationMeta{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: total,
			TotalPages: totalPages,
		},
	}, nil
}
