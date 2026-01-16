package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	consumerhandlers "challenge/cmd/consumers/handlers"
	"challenge/internal/audit"
	"challenge/internal/events"
	"challenge/internal/notification"
	"challenge/internal/payment"
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
	store := db.New()
	mockDB, err := db.NewMockClient()
	if err != nil {
		logger.Error("db init error", "error", err.Error())
		return
	}
	walletRepo := wallet.NewSQLRepository(mockDB)
	walletSvc := wallet.NewServiceWithRepo(walletRepo, metricsKit)
	paymentRepo := payment.NewSQLRepository(mockDB)
	paymentSvc := payment.NewService(bus, store, paymentRepo, metricsKit)
	gateway := external_payment_gateway.NewFakeGateway()
	recoverySvc := recovery.NewService(logger)
	auditSvc := audit.NewService(logger)
	notificationSvc := notification.NewService(logger)

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
	bus.Subscribe((events.WalletDebited{}).Name(), auditHandler.HandleAny)
	bus.Subscribe((events.WalletRefunded{}).Name(), auditHandler.HandleAny)
	bus.Subscribe((events.RecoveryRequested{}).Name(), auditHandler.HandleAny)
	bus.Subscribe((events.PaymentSucceeded{}).Name(), auditHandler.HandleAny)
	bus.Subscribe((events.PaymentFailed{}).Name(), auditHandler.HandleAny)

	bus.Subscribe((events.PaymentCreated{}).Name(), metricsHandler.HandleAny)
	bus.Subscribe((events.WalletDebited{}).Name(), metricsHandler.HandleAny)
	bus.Subscribe((events.WalletRefunded{}).Name(), metricsHandler.HandleAny)
	bus.Subscribe((events.PaymentSucceeded{}).Name(), metricsHandler.HandleAny)
	bus.Subscribe((events.PaymentFailed{}).Name(), metricsHandler.HandleAny)

	bus.Subscribe((events.PaymentSucceeded{}).Name(), notificationHandler.HandlePaymentCompleted)
	bus.Subscribe((events.PaymentFailed{}).Name(), notificationHandler.HandlePaymentFailed)

	bus.Subscribe((events.WalletDebited{}).Name(), walletHandler.HandleWalletDebited)
	bus.Subscribe((events.WalletRefunded{}).Name(), walletHandler.HandleWalletRefunded)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info("consumers started")
	<-ctx.Done()
	logger.Info("consumers stopped")
}
