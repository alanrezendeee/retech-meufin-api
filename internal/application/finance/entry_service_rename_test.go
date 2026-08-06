package finance

import (
	"context"
	"testing"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/finance"
)

// Renomear alcança TODAS as parcelas, inclusive as pagas — é o ponto da
// operação. A edição por lançamento só propaga para previstas futuras e
// deixaria a mesma dívida escrita de duas formas no histórico.
func TestRenameInstallmentGroupAtingeParcelasPagas(t *testing.T) {
	repo := newFakeEntryRepo()
	ws, groupID := seedInstallmentDebt(repo, 12, 5, 0, 50_000)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	novo := "Financiamento do carro (renegociado)"
	res, err := svc.RenameInstallmentGroup(context.Background(), ws, groupID, novo)
	if err != nil {
		t.Fatalf("RenameInstallmentGroup: %v", err)
	}
	if res.Entries != 12 {
		t.Errorf("parcelas renomeadas = %d, quer 12", res.Entries)
	}
	pagas := 0
	for _, e := range repo.entries {
		if e.RecurrenceGroupID == nil || *e.RecurrenceGroupID != groupID {
			continue
		}
		if e.Description != novo {
			t.Fatalf("parcela %v ficou com a descrição antiga: %q", e.InstallmentNumber, e.Description)
		}
		if e.Status == dom.StatusRealizada {
			pagas++
		}
	}
	if pagas != 5 {
		t.Errorf("parcelas pagas encontradas = %d, quer 5", pagas)
	}
}

// Residuais têm nome derivado ("Residual de X"); renomear a dívida sem
// renomeá-los deixaria o nome antigo pendurado na tela.
func TestRenameInstallmentGroupRenomeiaResiduais(t *testing.T) {
	repo := newFakeEntryRepo()
	ws, groupID := seedInstallmentDebt(repo, 12, 5, 3, 50_000)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	novo := "Dívida repactuada"
	res, err := svc.RenameInstallmentGroup(context.Background(), ws, groupID, novo)
	if err != nil {
		t.Fatalf("RenameInstallmentGroup: %v", err)
	}
	if res.Residuals != 3 {
		t.Fatalf("residuais renomeados = %d, quer 3", res.Residuals)
	}
	for _, e := range repo.entries {
		if e.ResidualOfID == nil {
			continue
		}
		if e.Description != dom.ResidualPrefix+novo {
			t.Fatalf("residual = %q, quer %q", e.Description, dom.ResidualPrefix+novo)
		}
	}
}

// Residual renomeado à mão pelo usuário é escolha dele — não pode ser
// sobrescrito pelo rename da série.
func TestRenameInstallmentGroupPreservaResidualRenomeadoManualmente(t *testing.T) {
	repo := newFakeEntryRepo()
	ws, groupID := seedInstallmentDebt(repo, 12, 5, 1, 50_000)
	manual := "Acordo do mês de março"
	var residualID uuid.UUID
	for _, e := range repo.entries {
		if e.ResidualOfID != nil {
			e.Description = manual
			residualID = e.ID
		}
	}
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	res, err := svc.RenameInstallmentGroup(context.Background(), ws, groupID, "Outro nome")
	if err != nil {
		t.Fatalf("RenameInstallmentGroup: %v", err)
	}
	if res.Residuals != 0 {
		t.Errorf("residual renomeado à mão não deveria ser tocado, veio %d", res.Residuals)
	}
	if repo.entries[residualID].Description != manual {
		t.Errorf("descrição manual foi sobrescrita: %q", repo.entries[residualID].Description)
	}
}

func TestRenameInstallmentGroupValida(t *testing.T) {
	repo := newFakeEntryRepo()
	ws, groupID := seedInstallmentDebt(repo, 12, 5, 0, 50_000)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	if _, err := svc.RenameInstallmentGroup(context.Background(), ws, groupID, "   "); err == nil {
		t.Error("descrição em branco deveria ser rejeitada")
	}
	if _, err := svc.RenameInstallmentGroup(context.Background(), ws, uuid.New(), "x"); err == nil {
		t.Error("grupo inexistente deveria devolver erro")
	}
}

// Renomear para o mesmo nome não faz escrita nenhuma.
func TestRenameInstallmentGroupNoOpQuandoIgual(t *testing.T) {
	repo := newFakeEntryRepo()
	ws, groupID := seedInstallmentDebt(repo, 12, 5, 0, 50_000)
	svc := NewFinancialEntryService(repo, fakeCategoryRepo{})

	res, err := svc.RenameInstallmentGroup(context.Background(), ws, groupID, "financiamento carro")
	if err != nil {
		t.Fatalf("RenameInstallmentGroup: %v", err)
	}
	if res.Entries != 0 || res.Residuals != 0 {
		t.Errorf("nome igual não deveria alterar nada, veio %d/%d", res.Entries, res.Residuals)
	}
}
