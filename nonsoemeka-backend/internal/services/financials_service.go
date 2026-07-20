package services

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"nonsoemeka-backend/internal/dto"
	"nonsoemeka-backend/internal/repository"
)

type FinancialsService interface {
	GetSummary(ctx context.Context, startDate, endDate string) (dto.FinancialSummaryResponse, error)
}

type financialsService struct {
	pool     *pgxpool.Pool
	saleRepo repository.SaleRepository
}

func NewFinancialsService(pool *pgxpool.Pool, saleRepo repository.SaleRepository) FinancialsService {
	return &financialsService{
		pool:     pool,
		saleRepo: saleRepo,
	}
}

func (s *financialsService) GetSummary(ctx context.Context, startDate, endDate string) (dto.FinancialSummaryResponse, error) {
	totalRev, totalCost, salesCount, itemsCount, err := s.saleRepo.GetFinancialSummary(ctx, s.pool, startDate, endDate)
	if err != nil {
		return dto.FinancialSummaryResponse{}, err
	}

	grossProfit := totalRev.Sub(totalCost)

	marginPct := decimal.Zero
	if !totalRev.IsZero() {
		marginPct = grossProfit.Div(totalRev).Mul(decimal.NewFromInt(100))
	}

	return dto.FinancialSummaryResponse{
		TotalRevenue:     totalRev.StringFixed(2),
		TotalCost:        totalCost.StringFixed(2),
		TotalGrossProfit: grossProfit.StringFixed(2),
		ProfitMarginPct:  marginPct.StringFixed(2),
		TotalSalesCount:  salesCount,
		TotalItemsSold:   itemsCount,
	}, nil
}
