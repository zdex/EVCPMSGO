package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"cpms/internal/config"
	"cpms/internal/db"
	"cpms/internal/models"
	"cpms/internal/repo"
	"cpms/internal/security"
)

func main() {
	id := flag.String("id", "CP-123", "chargePointId")
	secret := flag.String("secret", "devsecret", "shared secret (stored hashed)")
	active := flag.Bool("active", true, "mark charger active")
	vendor := flag.String("vendor", "ABB", "vendor")
	model := flag.String("model", "Terra54", "model")
	ocpp := siteName := flag.String("site", "", "optional site name (will be created if missing)")
	pricePerKwh := flag.Float64("price_per_kwh", 0, "optional per-kWh price for active tariff (requires --site)")
	currency := flag.String("currency", "USD", "tariff currency")
	flag.Parse()

	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	d, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()

	r := repo.NewChargersRepo(d.Pool)
	sites := repo.NewSitesRepo(d.Pool)
	tariffs := repo.NewTariffsRepo(d.Pool)

	hash := security.HashSecretSHA256(*secret)
	var siteId string
	if *siteName != "" {
		id, err := sites.Create(ctx, *siteName)
		if err != nil { log.Fatal(err) }
		siteId = id
		_ = r.SetSite(ctx, *id, siteId)
		if *pricePerKwh > 0 {
			_, err := tariffs.UpsertActiveForSite(ctx, siteId, *pricePerKwh, *currency)
			if err != nil { log.Fatal(err) }
		}
	}

	err = r.Upsert(ctx, models.Charger{
		ChargePointId: *id,
		SecretHash:    hash,
		IsActive:      *active,
		Vendor:        *vendor,
		Model:         *model,
		OcppVersion:   *ocpp,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Seeded charger:", *id, "active=", *active)
}
