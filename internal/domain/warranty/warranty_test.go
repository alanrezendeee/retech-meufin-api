package warranty

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func baseWarranty() *Warranty {
	return &Warranty{
		ID:                        uuid.New(),
		WorkspaceID:               uuid.New(),
		ItemName:                  "Geladeira Brastemp",
		Category:                  CategoryEletrodomestico,
		PurchaseDate:              time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		LegalWarrantyDays:         90,
		ContractualWarrantyMonths: 12,
		ExtendedWarrantyMonths:    0,
		Active:                    true,
	}
}

func TestExpiresAt_ContractualBeatsLegal(t *testing.T) {
	w := baseWarranty()
	// 12 meses > 90 dias => expira em 2027-01-01
	got := w.ExpiresAt()
	want := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("ExpiresAt = %v, want %v", got, want)
	}
}

func TestExpiresAt_ExtendedAddsToContractual(t *testing.T) {
	w := baseWarranty()
	w.ExtendedWarrantyMonths = 24 // 12 + 24 = 36 meses
	got := w.ExpiresAt()
	want := time.Date(2029, 1, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("ExpiresAt = %v, want %v", got, want)
	}
}

func TestExpiresAt_LegalBeatsShortContractual(t *testing.T) {
	w := baseWarranty()
	w.ContractualWarrantyMonths = 1 // 30 dias < 90 dias legais
	w.LegalWarrantyDays = 90
	got := w.ExpiresAt()
	want := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, 90)
	if !got.Equal(want) {
		t.Fatalf("ExpiresAt = %v, want %v", got, want)
	}
}

func TestStatusAt(t *testing.T) {
	w := baseWarranty() // expira 2027-01-01
	cases := []struct {
		name string
		ref  time.Time
		want Status
	}{
		{"vigente", time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), StatusVigente},
		{"expira_em_breve", time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC), StatusExpiraEmBreve},
		{"expirada", time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC), StatusExpirada},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := w.StatusAt(tc.ref); got != tc.want {
				t.Fatalf("StatusAt(%v) = %s, want %s", tc.ref, got, tc.want)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	w := baseWarranty()
	if err := w.Validate(); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
	w.ItemName = ""
	if err := w.Validate(); err == nil {
		t.Fatal("expected error for empty item_name")
	}
}
