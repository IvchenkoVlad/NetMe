package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/vladyslavivchenko/netme/internal/repositories"
	"github.com/vladyslavivchenko/netme/internal/services"
)

// Scheduler runs periodic background jobs using in-process tickers.
// Designed for single-instance MVP deployments; replace with a distributed
// queue (e.g. Asynq) when running multiple replicas.
type Scheduler struct {
	plaidSvc  *services.PlaidService
	itemRepo  *repositories.PlaidItemRepository
	acctRepo  *repositories.AccountRepository
	eventRepo *repositories.EventRepository
	log       *slog.Logger
}

func NewScheduler(
	plaidSvc *services.PlaidService,
	itemRepo *repositories.PlaidItemRepository,
	acctRepo *repositories.AccountRepository,
	eventRepo *repositories.EventRepository,
	log *slog.Logger,
) *Scheduler {
	return &Scheduler{
		plaidSvc:  plaidSvc,
		itemRepo:  itemRepo,
		acctRepo:  acctRepo,
		eventRepo: eventRepo,
		log:       log,
	}
}

// Start launches all background jobs and blocks until ctx is cancelled.
func (s *Scheduler) Start(ctx context.Context) {
	syncTicker := time.NewTicker(24 * time.Hour)
	snapshotTicker := time.NewTicker(24 * time.Hour)
	purgeTicker := time.NewTicker(7 * 24 * time.Hour)

	// Run once at startup so the first day doesn't have to wait.
	go s.runSync(ctx)
	go s.runNetWorthSnapshots(ctx)

	for {
		select {
		case <-ctx.Done():
			syncTicker.Stop()
			snapshotTicker.Stop()
			purgeTicker.Stop()
			s.log.Info("scheduler stopped")
			return
		case <-syncTicker.C:
			go s.runSync(ctx)
		case <-snapshotTicker.C:
			go s.runNetWorthSnapshots(ctx)
		case <-purgeTicker.C:
			go s.runDataRetentionPurge()
		}
	}
}

func (s *Scheduler) runSync(ctx context.Context) {
	userIDs, err := s.itemRepo.GetAllUserIDsWithItems()
	if err != nil {
		s.log.Error("daily sync: failed to load users", "err", err)
		return
	}
	s.log.Info("daily sync: starting", "users", len(userIDs))

	added := 0
	for _, uid := range userIDs {
		n, err := s.plaidSvc.SyncForUser(ctx, uid)
		if err != nil {
			s.log.Error("daily sync: user failed", "user_id", uid, "err", err)
			continue
		}
		added += n
	}
	s.log.Info("daily sync: done", "users", len(userIDs), "transactions_added", added)
}

func (s *Scheduler) runDataRetentionPurge() {
	n, err := s.eventRepo.PurgeOldRawEvents(90)
	if err != nil {
		s.log.Error("data retention purge: failed", "err", err)
		return
	}
	s.log.Info("data retention purge: done", "rows_deleted", n)
}

func (s *Scheduler) runNetWorthSnapshots(ctx context.Context) {
	userIDs, err := s.itemRepo.GetAllUserIDsWithItems()
	if err != nil {
		s.log.Error("net worth snapshot: failed to load users", "err", err)
		return
	}
	s.log.Info("net worth snapshot: starting", "users", len(userIDs))

	failed := 0
	for _, uid := range userIDs {
		if err := s.acctRepo.TakeNetWorthSnapshot(uid); err != nil {
			s.log.Error("net worth snapshot: user failed", "user_id", uid, "err", err)
			failed++
		}
	}
	s.log.Info("net worth snapshot: done", "users", len(userIDs), "failed", failed)
}
