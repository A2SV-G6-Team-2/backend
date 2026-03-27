package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	httpdelivery "expense_tracker/delivery/http"
	"expense_tracker/infrastructure/auth"
	"expense_tracker/infrastructure/db"
	infrarepo "expense_tracker/infrastructure/repository"
	"expense_tracker/infrastructure/repositoryPG"
	"expense_tracker/usecases"
)

func main() {
	log.Println("Starting Expense Tracker server...")

	if err := godotenv.Load(); err == nil {
		log.Println("Loaded .env file")
	}

	if err := db.DB_Init(); err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	userRepo := repositoryPG.NewUserRepoPG(db.DB)
	refreshTokenRepo := repositoryPG.NewRefreshTokenRepoPG(db.DB)
	expenseRepo := infrarepo.NewExpenseRepoPG(db.DB)
	debtReportRepo := repositoryPG.NewDebtRepoPG(db.DB)
	debtRepo := infrarepo.NewDebtRepositoryPG(db.DB)
	categoryRepo := infrarepo.NewCategoryRepoPG(db.DB)

	hasher := auth.BcryptHasher{}
	jwtSvc := auth.NewJWTService(os.Getenv("JWT_SECRET"))

	authUC := usecases.NewAuthUsecase(userRepo, refreshTokenRepo, hasher, jwtSvc)
	userUC := usecases.NewUserUsecase(userRepo)
	reportUC := usecases.NewReportUsecase(expenseRepo, debtReportRepo)
	debtUsecase := usecases.NewDebtUsecase(debtRepo)
	expenseUC := usecases.NewExpenseUseCase(expenseRepo)
	categoryUC := usecases.NewCategoryUseCase(categoryRepo)

	authHandler := httpdelivery.NewAuthHandler(authUC)
	userHandler := httpdelivery.NewUserHandler(userUC, jwtSvc)
	reportHandler := httpdelivery.NewReportHandler(reportUC, jwtSvc)
	debtHandler := httpdelivery.NewDebtHandler(debtUsecase, jwtSvc)
	expenseHandler := httpdelivery.NewExpenseHandler(expenseUC)
	categoryHandler := httpdelivery.NewCategoryHandler(categoryUC)

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/register", authHandler.Register)
	mux.HandleFunc("/auth/login", authHandler.Login)
	mux.HandleFunc("/auth/refresh", authHandler.Refresh)
	mux.HandleFunc("/auth/logout", authHandler.Logout)
	mux.HandleFunc("/user/profile", userHandler.GetProfile)
	mux.HandleFunc("/user/update", userHandler.UpdateProfile)
	mux.HandleFunc("/reports/weekly", reportHandler.GetWeeklyReport)
	mux.HandleFunc("/reports/daily", reportHandler.GetDailyReport)
	mux.HandleFunc("/reports/monthly", reportHandler.GetMonthlyReport)
	httpdelivery.RegisterDebtRoutes(mux, debtHandler)
	httpdelivery.RegisterExpenseRoutes(mux, expenseHandler)
	httpdelivery.RegisterCategoryRoutes(mux, categoryHandler)
	httpdelivery.ServeAPIDocs(mux)

	// JWT auth for /expenses and /categories; other routes unchanged
	handler := httpdelivery.JWTAuthMiddleware(jwtSvc, mux)

	log.Println("Server started on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
