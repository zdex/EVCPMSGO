package services

import (
	"context"

	"cpms/internal/repo"
)

type SettlementService struct {
	Chargers    *repo.ChargersRepo
	Sites       *repo.SitesRepo
	Sessions    *repo.SessionsRepo
	Settlements *repo.SettlementsRepo
}

// CreatePendingFromSession creates a Pending settlement once a session is priced.
// Idempotent: 1 settlement per session (unique(session_id)).
func (s *SettlementService) CreatePendingFromSession(ctx context.Context, sessionId string) error {
	sess, err := s.Sessions.GetByID(ctx, sessionId)
	if err != nil || sess == nil {
		return err
	}
	if sess.CostAmount == nil || sess.CostCurrency == nil {
		return nil
	}

	siteId, err := s.Chargers.GetSiteID(ctx, sess.ChargePointId)
	if err != nil || siteId == "" {
		return err
	}

	// Optional: ensure site has payout wallet configured (not mandatory for creating Pending).
	_, _ = s.Sites.GetPayoutWallet(ctx, siteId)

	_, err = s.Settlements.CreateForSession(ctx, sessionId, siteId, *sess.CostAmount, *sess.CostCurrency)
	return err
}
