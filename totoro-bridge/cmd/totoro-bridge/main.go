package main

import (
	"log"
	"os"

	"totoro-bridge/internal/bridgeapi"
	"totoro-bridge/internal/store"
)

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	addr := getenv("TOTOTO_BRIDGE_ADDR", ":18090")
	dbPath := getenv("TOTOTO_BRIDGE_DB", "./bridge.db")

	st, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}
	defer st.Close()

	r := bridgeapi.NewRouter(st, bridgeapi.Options{
		AdminKey: getenv("TOTOTO_BRIDGE_ADMIN_KEY", ""),
	})

	log.Printf("totoro-bridge listening on %s (db=%s)", addr, dbPath)
	if err := r.Run(addr); err != nil {
		log.Fatalf("run: %v", err)
	}
}
