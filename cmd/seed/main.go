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
	ocpp := flag.String("ocpp", "1.6J", "ocpp version")
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

	hash := security.HashSecretSHA256(*secret)
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
