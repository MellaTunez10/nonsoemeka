package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shopspring/decimal"
	"nonsoemeka-backend/internal/auth"
	"nonsoemeka-backend/internal/config"
	"nonsoemeka-backend/internal/database"
	"nonsoemeka-backend/internal/handlers"
	"nonsoemeka-backend/internal/middleware"
	"nonsoemeka-backend/internal/models"
	"nonsoemeka-backend/internal/repository"
	"nonsoemeka-backend/internal/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config initialization failed: %v\n", err)
		os.Exit(1)
	}

	var logHandler slog.Handler
	if cfg.Logging.Format == "json" {
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLogLevel(cfg.Logging.Level)})
	} else {
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: parseLogLevel(cfg.Logging.Level)})
	}
	slog.SetDefault(slog.New(logHandler))

	slog.Info("starting Nonsoemeka Pharmacy API server", "port", cfg.Server.Port)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx, cfg.Database)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := database.RunMigrations(ctx, pool, "migrations"); err != nil {
		slog.Error("database migration failed", "error", err)
		os.Exit(1)
	}

	// Repositories
	userRepo := repository.NewUserRepository()
	productRepo := repository.NewProductRepository()
	batchRepo := repository.NewBatchRepository()
	movementRepo := repository.NewInventoryMovementRepository()
	saleRepo := repository.NewSaleRepository()
	settingsRepo := repository.NewSettingsRepository()
	auditRepo := repository.NewAuditRepository()

	// Seed Data
	if err := seedInitialData(ctx, pool, userRepo, productRepo, batchRepo, movementRepo, settingsRepo, auditRepo); err != nil {
		slog.Warn("seed data check/execution error", "error", err)
	}

	// Services
	authService := services.NewAuthService(pool, userRepo, cfg)
	inventoryService := services.NewInventoryService(pool, productRepo, batchRepo, movementRepo, settingsRepo, auditRepo)
	checkoutService := services.NewCheckoutService(pool, saleRepo, batchRepo, productRepo, movementRepo, settingsRepo, userRepo)
	financialsService := services.NewFinancialsService(pool, saleRepo)
	reportsService := services.NewReportsService(pool, saleRepo)
	settingsService := services.NewSettingsService(pool, settingsRepo, auditRepo)
	staffService := services.NewStaffManagementService(pool, userRepo, auditRepo)

	// Handlers
	authHandler := handlers.NewAuthHandler(authService, cfg)
	inventoryHandler := handlers.NewInventoryHandler(inventoryService)
	checkoutHandler := handlers.NewCheckoutHandler(checkoutService)
	financialsHandler := handlers.NewFinancialsHandler(financialsService)
	reportsHandler := handlers.NewReportsHandler(reportsService)
	settingsHandler := handlers.NewSettingsHandler(settingsService)
	staffHandler := handlers.NewStaffManagementHandler(staffService)

	// Rate limiters
	globalLimiter := middleware.NewRateLimiter(cfg.RateLimit.GlobalPerMinute, time.Minute)
	loginLimiter := middleware.NewRateLimiter(cfg.RateLimit.LoginPerMinute, time.Minute)

	// Router wiring
	r := chi.NewRouter()

	r.Use(middleware.RecoveryMiddleware)
	r.Use(middleware.RequestIDMiddleware)
	r.Use(middleware.LoggingAndMetricsMiddleware)
	r.Use(middleware.CORSMiddleware(cfg.AllowedOrigins))
	r.Use(middleware.TimeoutMiddleware(cfg.Server.ReadTimeout))
	r.Use(middleware.RateLimitMiddleware(globalLimiter))

	// --- Public endpoints ---
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("UNREADY"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("READY"))
	})
	r.Handle("/metrics", promhttp.Handler())

	r.Route("/api/v1/auth", func(r chi.Router) {
		r.With(middleware.RateLimitMiddleware(loginLimiter)).Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.Refresh)
		r.Post("/logout", authHandler.Logout)
	})

	// Protected routes (JWT mandatory)
	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(cfg.JWT.AccessSecret))

		// --- Staff and Admin Endpoints ---
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRoles(string(models.RoleAdmin), string(models.RoleStaff)))
			r.Get("/api/v1/products", inventoryHandler.ListProducts)
			r.Post("/api/v1/checkout", checkoutHandler.ProcessCheckout)
		})

		// --- Admin-Only Endpoints ---
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireRoles(string(models.RoleAdmin)))

			// Inventory Management
			r.Post("/api/v1/admin/inventory/products", inventoryHandler.CreateProduct)
			r.Get("/api/v1/admin/inventory/products", inventoryHandler.ListProducts)
			r.Post("/api/v1/admin/inventory/batches", inventoryHandler.RegisterBatch)
			r.Get("/api/v1/admin/inventory/batches", inventoryHandler.ListBatches)
			r.Post("/api/v1/admin/inventory/batches/{id}/adjust", inventoryHandler.AdjustStock)
			r.Post("/api/v1/admin/inventory/batches/{id}/write-off", inventoryHandler.WriteOffStock)
			r.Get("/api/v1/admin/inventory/expiry", inventoryHandler.ListExpiringBatches)
			r.Get("/api/v1/admin/inventory/movements", inventoryHandler.ListMovements)

			// Financials
			r.Get("/api/v1/admin/financials/summary", financialsHandler.GetSummary)

			// Reports & Analytics
			r.Get("/api/v1/admin/reports/sales-trends", reportsHandler.GetSalesTrends)
			r.Get("/api/v1/admin/reports/top-products", reportsHandler.GetTopProducts)

			// Staff Management & Audit Logs
			r.Post("/api/v1/admin/staff", staffHandler.CreateStaff)
			r.Get("/api/v1/admin/staff", staffHandler.ListStaff)
			r.Put("/api/v1/admin/staff/{id}", staffHandler.UpdateStaff)
			r.Get("/api/v1/admin/audit-logs", staffHandler.ListAuditLogs)

			// Settings
			r.Get("/api/v1/admin/settings", settingsHandler.GetSettings)
			r.Put("/api/v1/admin/settings", settingsHandler.UpdateSettings)
		})
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		slog.Info("shutting down server gracefully...")

		shutdownCtx, cancelShutdown := context.WithTimeout(serverCtx, cfg.Server.ShutdownTimeout)
		defer cancelShutdown()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("server shutdown forced", "error", err)
		}
		serverStopCtx()
	}()

	slog.Info("server listening", "address", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}

	<-serverCtx.Done()
	slog.Info("server exited cleanly")
}

func parseLogLevel(levelStr string) slog.Level {
	switch levelStr {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func seedInitialData(
	ctx context.Context,
	pool repository.DBTX,
	userRepo repository.UserRepository,
	productRepo repository.ProductRepository,
	batchRepo repository.BatchRepository,
	movementRepo repository.InventoryMovementRepository,
	settingsRepo repository.SettingsRepository,
	auditRepo repository.AuditRepository,
) error {
	_, totalUsers, err := userRepo.List(ctx, pool, 1, 1)
	if err == nil && totalUsers > 0 {
		slog.Info("seed data already exists, skipping initial seed")
		return nil
	}

	slog.Info("seeding initial database records...")

	// Seed Settings
	defaultMarkupBytes, _ := json.Marshal("25.00")
	expiryDaysBytes, _ := json.Marshal(30)
	lowStockBytes, _ := json.Marshal(10)
	pharmacyNameBytes, _ := json.Marshal("Nonsoemeka Pharmacy")
	receiptFooterBytes, _ := json.Marshal("Thank you for trusting Nonsoemeka Pharmacy! Wish you good health.")

	_ = settingsRepo.Set(ctx, pool, "default_markup_percentage", defaultMarkupBytes, nil)
	_ = settingsRepo.Set(ctx, pool, "expiry_alert_days", expiryDaysBytes, nil)
	_ = settingsRepo.Set(ctx, pool, "low_stock_threshold", lowStockBytes, nil)
	_ = settingsRepo.Set(ctx, pool, "pharmacy_name", pharmacyNameBytes, nil)
	_ = settingsRepo.Set(ctx, pool, "receipt_footer", receiptFooterBytes, nil)

	// Seed Admin & Staff Users
	adminPassHash, _ := auth.HashPassword("AdminPass123!")
	staffPassHash, _ := auth.HashPassword("StaffPass123!")

	adminUser, err := userRepo.Create(ctx, pool, models.User{
		Username:     "admin",
		Email:        "admin@nonsoemeka.com",
		PasswordHash: adminPassHash,
		Role:         models.RoleAdmin,
		IsActive:     true,
	})
	if err != nil {
		return fmt.Errorf("failed to seed admin user: %w", err)
	}

	staffUser, err := userRepo.Create(ctx, pool, models.User{
		Username:     "staff",
		Email:        "staff@nonsoemeka.com",
		PasswordHash: staffPassHash,
		Role:         models.RoleStaff,
		IsActive:     true,
	})
	if err != nil {
		return fmt.Errorf("failed to seed staff user: %w", err)
	}

	// Seed Products
	desc1 := "Pain reliever and fever reducer 500mg tablets"
	p1, _ := productRepo.Create(ctx, pool, models.Product{Name: "Paracetamol 500mg", SKU: "PARA-500", Description: &desc1, IsActive: true})

	desc2 := "Broad-spectrum antibiotic 500mg capsules"
	p2, _ := productRepo.Create(ctx, pool, models.Product{Name: "Amoxicillin 500mg", SKU: "AMOX-500", Description: &desc2, IsActive: true})

	desc3 := "Nonsteroidal anti-inflammatory drug 400mg"
	p3, _ := productRepo.Create(ctx, pool, models.Product{Name: "Ibuprofen 400mg", SKU: "IBU-400", Description: &desc3, IsActive: true})

	desc4 := "Immune system support 1000mg effervescent"
	p4, _ := productRepo.Create(ctx, pool, models.Product{Name: "Vitamin C 1000mg", SKU: "VITC-1000", Description: &desc4, IsActive: true})

	now := time.Now()

	// Seed Batches
	b1, _ := batchRepo.Create(ctx, pool, models.Batch{
		ProductID:         p1.ID,
		BatchNumber:       "BATCH-PARA-001",
		QuantityReceived:  200,
		QuantityRemaining: 200,
		ExpiryDate:        now.AddDate(0, 6, 0),
		CostPrice:         decimal.NewFromFloat(50.00),
		MarkupPercentage:  decimal.NewFromFloat(25.00),
	})
	_ = movementRepo.Create(ctx, pool, models.InventoryMovement{BatchID: b1.ID, MovementType: models.MovementReceived, QuantityDelta: 200, CreatedBy: adminUser.ID})

	b2, _ := batchRepo.Create(ctx, pool, models.Batch{
		ProductID:         p1.ID,
		BatchNumber:       "BATCH-PARA-002",
		QuantityReceived:  150,
		QuantityRemaining: 150,
		ExpiryDate:        now.AddDate(0, 3, 0), // Expiring sooner -> FEFO priority
		CostPrice:         decimal.NewFromFloat(48.00),
		MarkupPercentage:  decimal.NewFromFloat(30.00),
	})
	_ = movementRepo.Create(ctx, pool, models.InventoryMovement{BatchID: b2.ID, MovementType: models.MovementReceived, QuantityDelta: 150, CreatedBy: adminUser.ID})

	b3, _ := batchRepo.Create(ctx, pool, models.Batch{
		ProductID:         p2.ID,
		BatchNumber:       "BATCH-AMOX-001",
		QuantityReceived:  100,
		QuantityRemaining: 80,
		ExpiryDate:        now.AddDate(0, 12, 0),
		CostPrice:         decimal.NewFromFloat(120.00),
		MarkupPercentage:  decimal.NewFromFloat(25.00),
	})
	_ = movementRepo.Create(ctx, pool, models.InventoryMovement{BatchID: b3.ID, MovementType: models.MovementReceived, QuantityDelta: 100, CreatedBy: adminUser.ID})

	// Add an adjustment movement to b3
	reasonAdj := "Damaged packaging during shelf stocking"
	_ = movementRepo.Create(ctx, pool, models.InventoryMovement{BatchID: b3.ID, MovementType: models.MovementAdjustment, QuantityDelta: -20, Reason: &reasonAdj, CreatedBy: adminUser.ID})

	b4, _ := batchRepo.Create(ctx, pool, models.Batch{
		ProductID:         p3.ID,
		BatchNumber:       "BATCH-IBU-EXP",
		QuantityReceived:  50,
		QuantityRemaining: 0,
		ExpiryDate:        now.AddDate(0, 0, -10), // Expired batch
		CostPrice:         decimal.NewFromFloat(80.00),
		MarkupPercentage:  decimal.NewFromFloat(25.00),
	})
	_ = movementRepo.Create(ctx, pool, models.InventoryMovement{BatchID: b4.ID, MovementType: models.MovementReceived, QuantityDelta: 50, CreatedBy: adminUser.ID})

	reasonWriteOff := "Batch passed expiration date on shelf"
	_ = movementRepo.Create(ctx, pool, models.InventoryMovement{BatchID: b4.ID, MovementType: models.MovementExpiredWriteOff, QuantityDelta: -50, Reason: &reasonWriteOff, CreatedBy: adminUser.ID})

	b5, _ := batchRepo.Create(ctx, pool, models.Batch{
		ProductID:         p4.ID,
		BatchNumber:       "BATCH-VITC-001",
		QuantityReceived:  300,
		QuantityRemaining: 300,
		ExpiryDate:        now.AddDate(1, 0, 0),
		CostPrice:         decimal.NewFromFloat(35.00),
		MarkupPercentage:  decimal.NewFromFloat(40.00),
	})
	_ = movementRepo.Create(ctx, pool, models.InventoryMovement{BatchID: b5.ID, MovementType: models.MovementReceived, QuantityDelta: 300, CreatedBy: adminUser.ID})

	_ = staffUser
	slog.Info("initial database seeding completed successfully")
	return nil
}
