package target

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSQLiteStore_AllOperations(t *testing.T) {
	// 1. Setup temporary database file
	tempDir, err := os.MkdirTemp("", "trading_alert_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize SQLiteStore: %v", err)
	}
	defer store.Close()

	// 2. Test standard price target validation and save
	tPrice, err := store.Save(Target{
		BotName:     "bot-1",
		Symbol:      "BTCUSDT",
		Type:        TargetTypePrice,
		TargetPrice: 65000.0,
		Direction:   DirectionUp,
		Note:        "Standard Target Note",
	})
	if err != nil {
		t.Fatalf("failed to save standard price target: %v", err)
	}

	if tPrice.ID == "" {
		t.Error("expected generated target ID, got empty string")
	}
	if tPrice.Symbol != "BTCUSDT" {
		t.Errorf("expected symbol BTCUSDT, got %s", tPrice.Symbol)
	}

	// 3. Test trailing target validation and save
	tTrailing, err := store.Save(Target{
		BotName:         "bot-2",
		Symbol:          "ethusdt",
		Type:            TargetTypeTrailing,
		TrailingPercent: 5.0,
		Direction:       DirectionDown,
		ActivationPrice: 3500.0,
		Note:            "Trailing Target Note",
	})
	if err != nil {
		t.Fatalf("failed to save trailing target: %v", err)
	}

	if tTrailing.IsActive {
		t.Error("expected trailing target with activation price to be inactive on creation")
	}

	// 4. Test Get target
	retrieved, err := store.Get(tPrice.ID)
	if err != nil {
		t.Fatalf("failed to get target: %v", err)
	}
	if retrieved.Note != "Standard Target Note" {
		t.Errorf("expected Note 'Standard Target Note', got: %s", retrieved.Note)
	}
	if retrieved.Type != TargetTypePrice {
		t.Errorf("expected Type 'price', got: %s", retrieved.Type)
	}

	// Test Get not found
	_, err = store.Get("non-existent-id")
	if err == nil {
		t.Error("expected error getting non-existent target, got nil")
	}

	// 5. Test ListAll
	allTargets, err := store.ListAll()
	if err != nil {
		t.Fatalf("failed to list all targets: %v", err)
	}
	if len(allTargets) != 2 {
		t.Errorf("expected 2 targets, got %d", len(allTargets))
	}

	// 6. Test ListBySymbol
	btcTargets, err := store.ListBySymbol("BTCUSDT")
	if err != nil {
		t.Fatalf("failed to list by symbol: %v", err)
	}
	if len(btcTargets) != 1 {
		t.Errorf("expected 1 BTCUSDT target, got %d", len(btcTargets))
	}
	if btcTargets[0].ID != tPrice.ID {
		t.Errorf("expected target ID %s, got %s", tPrice.ID, btcTargets[0].ID)
	}

	// Case-insensitive list checking
	ethTargets, err := store.ListBySymbol("ethusdt")
	if err != nil {
		t.Fatalf("failed to list by symbol case-insensitive: %v", err)
	}
	if len(ethTargets) != 1 {
		t.Errorf("expected 1 ETHUSDT target, got %d", len(ethTargets))
	}

	// 7. Test ListByBotName
	bot1Targets, err := store.ListByBotName("bot-1")
	if err != nil {
		t.Fatalf("failed to list by bot name: %v", err)
	}
	if len(bot1Targets) != 1 {
		t.Errorf("expected 1 target for bot-1, got %d", len(bot1Targets))
	}

	// 8. Test Update
	retrieved.LastState = StateAbove
	retrieved.Note = "Updated Note"
	err = store.Update(retrieved)
	if err != nil {
		t.Fatalf("failed to update target: %v", err)
	}

	updatedTarget, err := store.Get(tPrice.ID)
	if err != nil {
		t.Fatalf("failed to get updated target: %v", err)
	}
	if updatedTarget.LastState != StateAbove {
		t.Errorf("expected updated state 'above', got %s", updatedTarget.LastState)
	}
	if updatedTarget.Note != "Updated Note" {
		t.Errorf("expected updated note 'Updated Note', got %s", updatedTarget.Note)
	}

	// 9. Test Delete
	err = store.Delete(tPrice.ID)
	if err != nil {
		t.Fatalf("failed to delete target: %v", err)
	}

	_, err = store.Get(tPrice.ID)
	if err == nil || !strings.Contains(err.Error(), "target not found") {
		t.Errorf("expected 'target not found' error, got %v", err)
	}

	// Verify count is now 1
	allTargets, _ = store.ListAll()
	if len(allTargets) != 1 {
		t.Errorf("expected 1 target left, got %d", len(allTargets))
	}
}
