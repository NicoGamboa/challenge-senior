package main

import (
	"context"
	"net/http"
	"time"

	consumerhandlers "challenge/cmd/consumers/handlers"
	"challenge/cmd/web/handlers"
	"challenge/cmd/web/validator"
	"challenge/internal/audit"
	"challenge/internal/events"
	"challenge/internal/health"
	"challenge/internal/notification"
	"challenge/internal/payment"
	"challenge/internal/readmodels"
	"challenge/internal/recovery"
	"challenge/internal/wallet"
	"challenge/kit/broker"
	"challenge/kit/db"
	"challenge/kit/external_payment_gateway"
	"challenge/kit/observability"
)

func main() {
	logger := observability.NewLogger()
	metricsKit := observability.NewMetrics()
	bus := broker.New()
	defer bus.Close()
	store, err := db.NewWithFile("./out/db.jsonl")
	if err != nil {
		logger.Error("db init error", "error", err.Error())
		return
	}
	defer func() { _ = store.Close() }()

	auditSvc, err := audit.NewServiceWithFile(logger, "./out/audit.jsonl")
	if err != nil {
		logger.Error("audit init error", "error", err.Error())
		return
	}
	defer func() { _ = auditSvc.Close() }()

	recoverySvc := recovery.NewService(logger)
	notificationSvc := notification.NewService(logger)
	gateway := external_payment_gateway.NewCircuitBreakerGateway(
		external_payment_gateway.NewFakeGateway(),
		external_payment_gateway.CircuitBreakerConfig{
			FailureThreshold: 3,
			SuccessThreshold: 1,
			OpenTimeout:      2 * time.Second,
		},
	)
	mockDB, err := db.NewMockClient(
		db.WithWalletsJSONFile("./out/wallets.json"),
		db.WithWalletsJSONPersistence("./out/wallets.json"),
	)
	if err != nil {
		logger.Error("db init error", "error", err.Error())
		return
	}
	walletRepo := wallet.NewSQLRepository(mockDB)
	walletSvc := wallet.NewServiceWithRepo(walletRepo, metricsKit)
	paymentRepo := payment.NewSQLRepository(mockDB)
	paymentSvc := payment.NewService(bus, store, paymentRepo, metricsKit)
	projector := readmodels.NewProjector()
	if err := projector.Replay(context.Background(), store); err != nil {
		logger.Error("read model replay error", "error", err.Error())
		return
	}
	healthSvc := health.NewService(2*time.Second, map[string]health.CheckFunc{
		"db": func(ctx context.Context) error {
			row, err := mockDB.QueryRow(ctx, "SELECT balance FROM wallets WHERE user_id = ?", "__healthcheck__")
			if err != nil {
				return err
			}
			var bal int64
			if err := row.Scan(&bal); err != nil {
				if db.IsNotFound(err) {
					return nil
				}
				return err
			}
			return nil
		},
		"gateway": func(ctx context.Context) error {
			callCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			defer cancel()
			_, err := gateway.Charge(callCtx, "__healthcheck__", 1)
			return err
		},
	})
	jsonV := validator.NewJSON()

	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for range t.C {
			logger.Info(
				"metrics snapshot",
				"payments_created", metricsKit.PaymentsCreated.Load(),
				"payments_succeeded", metricsKit.PaymentsSucceeded.Load(),
				"payments_failed", metricsKit.PaymentsFailed.Load(),
				"wallet_debits", metricsKit.WalletDebits.Load(),
				"wallet_refunds", metricsKit.WalletRefunds.Load(),
			)
		}
	}()

	gatewayHandler := consumerhandlers.NewPaymentEvent(logger, bus, gateway, recoverySvc)
	resultHandler := consumerhandlers.NewPaymentResultEvent(logger, bus, paymentSvc)
	paymentFlowHandler := consumerhandlers.NewPaymentFlowEvent(logger, bus, paymentSvc)
	auditHandler := consumerhandlers.NewAuditEvent(auditSvc)
	walletHandler := consumerhandlers.NewWalletEvent(logger, bus, walletSvc)
	metricsHandler := consumerhandlers.NewMetricsEvent(metricsKit)
	notificationHandler := consumerhandlers.NewNotificationEvent(notificationSvc)
	recoveryEventHandler := consumerhandlers.NewRecoveryEvent(logger, bus, paymentSvc, time.Minute, nil)

	bus.Subscribe((events.PaymentChargeRequested{}).Name(), gatewayHandler.HandleChargeRequested)
	bus.Subscribe((events.PaymentChargeSucceeded{}).Name(), resultHandler.HandleChargeSucceeded)
	bus.Subscribe((events.PaymentChargeFailed{}).Name(), resultHandler.HandleChargeFailed)
	bus.Subscribe((events.RecoveryRequested{}).Name(), recoveryEventHandler.HandleRecoveryRequested)

	bus.Subscribe((events.PaymentInitialized{}).Name(), walletHandler.HandlePaymentInitialized)
	bus.Subscribe((events.WalletDebitRequested{}).Name(), walletHandler.HandleWalletDebitRequested)
	bus.Subscribe((events.WalletDebitRejected{}).Name(), paymentFlowHandler.HandleWalletDebitRejected)
	bus.Subscribe((events.WalletDebited{}).Name(), paymentFlowHandler.HandleWalletDebited)
	bus.Subscribe((events.WalletRefundRequested{}).Name(), walletHandler.HandleWalletRefundRequested)

	bus.Subscribe((events.PaymentCreated{}).Name(), auditHandler.HandleAny)
	bus.Subscribe((events.PaymentInitialized{}).Name(), auditHandler.HandleAny)
	bus.Subscribe((events.PaymentPending{}).Name(), auditHandler.HandleAny)
	bus.Subscribe((events.PaymentRejected{}).Name(), auditHandler.HandleAny)
	bus.Subscribe((events.WalletDebited{}).Name(), auditHandler.HandleAny)
	bus.Subscribe((events.WalletRefunded{}).Name(), auditHandler.HandleAny)
	bus.Subscribe((events.RecoveryRequested{}).Name(), auditHandler.HandleAny)
	bus.Subscribe((events.PaymentSucceeded{}).Name(), auditHandler.HandleAny)
	bus.Subscribe((events.PaymentFailed{}).Name(), auditHandler.HandleAny)

	bus.Subscribe((events.PaymentCreated{}).Name(), projector.Apply)
	bus.Subscribe((events.PaymentInitialized{}).Name(), projector.Apply)
	bus.Subscribe((events.PaymentPending{}).Name(), projector.Apply)
	bus.Subscribe((events.PaymentRejected{}).Name(), projector.Apply)
	bus.Subscribe((events.PaymentSucceeded{}).Name(), projector.Apply)
	bus.Subscribe((events.PaymentFailed{}).Name(), projector.Apply)
	bus.Subscribe((events.WalletCredited{}).Name(), projector.Apply)
	bus.Subscribe((events.WalletDebited{}).Name(), projector.Apply)
	bus.Subscribe((events.WalletRefunded{}).Name(), projector.Apply)

	bus.Subscribe((events.PaymentCreated{}).Name(), metricsHandler.HandleAny)
	bus.Subscribe((events.WalletDebited{}).Name(), metricsHandler.HandleAny)
	bus.Subscribe((events.WalletRefunded{}).Name(), metricsHandler.HandleAny)
	bus.Subscribe((events.PaymentSucceeded{}).Name(), metricsHandler.HandleAny)
	bus.Subscribe((events.PaymentFailed{}).Name(), metricsHandler.HandleAny)

	bus.Subscribe((events.PaymentSucceeded{}).Name(), notificationHandler.HandlePaymentCompleted)
	bus.Subscribe((events.PaymentFailed{}).Name(), notificationHandler.HandlePaymentFailed)

	bus.Subscribe((events.WalletDebited{}).Name(), walletHandler.HandleWalletDebited)
	bus.Subscribe((events.WalletRefunded{}).Name(), walletHandler.HandleWalletRefunded)

	walletH := handlers.NewWallet(jsonV, bus, store, walletSvc, projector)
	paymentH := handlers.NewPayment(jsonV, bus, store, paymentSvc, healthSvc, projector)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /wallet/credit", walletH.Credit)
	mux.HandleFunc("GET /wallet/", walletH.Balance)
	mux.HandleFunc("POST /payments", paymentH.Create)
	mux.HandleFunc("GET /payments/", paymentH.Get)

	srv := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 2 * time.Second}

	logger.Info("web server started", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("web server error", "error", err.Error())
	}
}
