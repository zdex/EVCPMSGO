package services

import (
	"context"
	"math"

	"cpms/internal/repo"
)

type PricingService struct {
	Chargers *repo.ChargersRepo
	Tariffs  *repo.TariffsRepo
	Sessions *repo.SessionsRepo
}

func NewPricingService(chargers *repo.ChargersRepo, tariffs *repo.TariffsRepo, sessions *repo.SessionsRepo) *PricingService {
	return &PricingService{Chargers: chargers, Tariffs: tariffs, Sessions: sessions}
}

// PriceSessionPerKwh computes cost using active site tariff:
// cost = (energy_wh/1000) * price_per_kwh
// If energy_wh is missing or no site/tariff, it does nothing (idempotent).
func (p *PricingService) PriceSessionPerKwh(ctx context.Context, sessionId string) error {
	sess, err := p.Sessions.GetByID(ctx, sessionId)
	if err != nil || sess == nil {
		return err
	}
	if sess.EnergyWh == nil {
		return nil
	}

	// Find charger site_id
	//var siteId *string
	// direct query through sessions repo's db pool (pgxpool) isn't exposed; simplest: use an internal query via SessionsRepo's db
	// We'll rely on a helper query on sessions repo (not added). Instead we use a join query here via SessionsRepo's db.
	// To keep repo boundaries, this service will be wired with TariffsRepo and a new helper on ChargersRepo would be ideal.
	// For MVP, we do a small join in TariffsRepo: active tariff by charger id.
	return p.priceByCharger(ctx, sess.ChargePointId, sessionId, *sess.EnergyWh)
}

func (p *PricingService) priceByCharger(ctx context.Context, chargePointId string, sessionId string, energyWh int64) error {
	// Fetch charger site_id (direct query through ChargersRepo db)
	// ChargersRepo currently doesn’t expose the pool; simplest: add a method on ChargersRepo to get site_id.
	siteId, err := p.Chargers.GetSiteID(ctx, chargePointId)
	if err != nil || siteId == "" {
		return err
	}
	tariff, err := p.Tariffs.GetActiveForSite(ctx, siteId)
	if err != nil || tariff == nil {
		return err
	}
	kwh := float64(energyWh) / 1000.0
	cost := kwh * tariff.PricePerKwh
	cost = round(cost, 4)
	return p.Sessions.SetPricing(ctx, sessionId, tariff.TariffId, cost, tariff.Currency)
}

func round(v float64, places int) float64 {
	pow := math.Pow(10, float64(places))
	return math.Round(v*pow) / pow
}
