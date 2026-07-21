package entitlement

import (
	"testing"

	"github.com/google/uuid"
)

func TestEffectiveFiscalSEFAZQuota_DefaultTierFree(t *testing.T) {
	e := Default(uuid.New())
	if e.Tier != TierFree {
		t.Fatalf("Default deveria ser tier free, veio %q", e.Tier)
	}
	if got, want := e.EffectiveFiscalSEFAZQuota(), DefaultFiscalSEFAZQuota(TierFree); got != want {
		t.Fatalf("cota free: got %d, want %d", got, want)
	}
}

func TestEffectiveFiscalSEFAZQuota_TierProDefault(t *testing.T) {
	e := &Entitlement{Tier: TierPro}
	if got, want := e.EffectiveFiscalSEFAZQuota(), DefaultFiscalSEFAZQuota(TierPro); got != want {
		t.Fatalf("cota pro: got %d, want %d", got, want)
	}
}

func TestEffectiveFiscalSEFAZQuota_OverrideVence(t *testing.T) {
	override := 999
	e := &Entitlement{Tier: TierFree, FiscalSEFAZQuota: &override}
	if got := e.EffectiveFiscalSEFAZQuota(); got != override {
		t.Fatalf("override deveria vencer o default do tier: got %d, want %d", got, override)
	}
}
