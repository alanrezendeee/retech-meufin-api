package health

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

const maxFamilyMemberNameLen = 255

// Relationship enumera o vínculo do familiar com o titular do workspace.
type Relationship string

const (
	RelationshipSelf   Relationship = "self"
	RelationshipSpouse Relationship = "spouse"
	RelationshipChild  Relationship = "child"
	RelationshipParent Relationship = "parent"
	RelationshipOther  Relationship = "other"
)

func validRelationships() map[Relationship]struct{} {
	return map[Relationship]struct{}{
		RelationshipSelf:   {},
		RelationshipSpouse: {},
		RelationshipChild:  {},
		RelationshipParent: {},
		RelationshipOther:  {},
	}
}

// FamilyMember representa um membro da família cujos dados de saúde são acompanhados.
type FamilyMember struct {
	ID           uuid.UUID
	WorkspaceID  uuid.UUID
	FullName     string
	Relationship string
	BirthDate    *time.Time
	Gender       *string
	Document     *string
	Notes        *string
	HeightCm     *float64
	WeightKg     *float64
	Active       bool
	// AvatarObjectKey aponta para a foto do membro no object storage (nil = sem foto).
	AvatarObjectKey *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Age retorna a idade atual em anos completos, ou nil se não houver data de nascimento.
func (f *FamilyMember) Age() *int {
	if f.BirthDate == nil {
		return nil
	}
	a := yearsBetween(*f.BirthDate, time.Now().UTC())
	return &a
}

// yearsBetween calcula anos completos entre birth e ref.
func yearsBetween(birth, ref time.Time) int {
	years := ref.Year() - birth.Year()
	// Ainda não fez aniversário neste ano.
	if ref.Month() < birth.Month() || (ref.Month() == birth.Month() && ref.Day() < birth.Day()) {
		years--
	}
	if years < 0 {
		years = 0
	}
	return years
}

// Birthday descreve o próximo aniversário de um membro (usado no quadro do painel).
type Birthday struct {
	Member       FamilyMember
	Age          int       // idade atual (anos completos)
	Turns        int       // idade que fará no próximo aniversário
	NextBirthday time.Time // data do próximo aniversário
	DaysUntil    int       // dias restantes até o próximo aniversário (0 = hoje)
}

// NextBirthdayOf calcula o próximo aniversário de um membro a partir de "ref".
// Retorna false se o membro não tem data de nascimento.
func NextBirthdayOf(f FamilyMember, ref time.Time) (Birthday, bool) {
	if f.BirthDate == nil {
		return Birthday{}, false
	}
	birth := f.BirthDate.UTC()
	// Normaliza a referência para meia-noite (contagem de dias por data-calendário).
	today := time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, time.UTC)

	// Próximo aniversário: mesmo dia/mês, começando pelo ano corrente.
	next := birthdayInYear(birth, today.Year())
	if next.Before(today) {
		next = birthdayInYear(birth, today.Year()+1)
	}
	days := int(next.Sub(today).Hours() / 24)
	age := yearsBetween(birth, today)
	return Birthday{
		Member:       f,
		Age:          age,
		Turns:        age + 1,
		NextBirthday: next,
		DaysUntil:    days,
	}, true
}

// birthdayInYear resolve a data do aniversário no ano informado, tratando 29/02
// em anos não bissextos (cai em 28/02).
func birthdayInYear(birth time.Time, year int) time.Time {
	month, day := birth.Month(), birth.Day()
	if month == time.February && day == 29 && !isLeap(year) {
		day = 28
	}
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func isLeap(year int) bool {
	return (year%4 == 0 && year%100 != 0) || year%400 == 0
}

func (f *FamilyMember) Validate() error {
	if f.WorkspaceID == uuid.Nil {
		return &ValidationError{Msg: "workspace_id é obrigatório"}
	}
	name := strings.TrimSpace(f.FullName)
	if name == "" {
		return &ValidationError{Msg: "nome completo é obrigatório"}
	}
	if len(name) > maxFamilyMemberNameLen {
		return &ValidationError{Msg: "nome completo excede o tamanho máximo"}
	}
	rel := Relationship(strings.TrimSpace(strings.ToLower(f.Relationship)))
	if _, ok := validRelationships()[rel]; !ok {
		return &ValidationError{Msg: "relationship inválido (self|spouse|child|parent|other)"}
	}
	f.FullName = name
	f.Relationship = string(rel)
	if f.Gender != nil {
		g := strings.TrimSpace(*f.Gender)
		if g == "" {
			f.Gender = nil
		} else {
			f.Gender = &g
		}
	}
	if f.Document != nil {
		d := strings.TrimSpace(*f.Document)
		if d == "" {
			f.Document = nil
		} else {
			f.Document = &d
		}
	}
	if f.Notes != nil {
		n := strings.TrimSpace(*f.Notes)
		if n == "" {
			f.Notes = nil
		} else {
			f.Notes = &n
		}
	}
	if f.HeightCm != nil && (*f.HeightCm <= 0 || *f.HeightCm > 300) {
		return &ValidationError{Msg: "altura (cm) fora do intervalo válido"}
	}
	if f.WeightKg != nil && (*f.WeightKg <= 0 || *f.WeightKg > 700) {
		return &ValidationError{Msg: "peso (kg) fora do intervalo válido"}
	}
	return nil
}

// FamilyMemberFilter recorta a listagem da tela de gestão.
type FamilyMemberFilter struct {
	Query        string // busca por nome (case-insensitive)
	Relationship string
	Active       *bool
}

// FamilyMemberRepository abstrai a persistência de membros da família (workspace-scoped).
type FamilyMemberRepository interface {
	Create(ctx context.Context, f *FamilyMember) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*FamilyMember, error)
	Update(ctx context.Context, f *FamilyMember) error
	SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error
	List(ctx context.Context, workspaceID uuid.UUID, filter FamilyMemberFilter, limit, offset int) ([]FamilyMember, int64, error)
	// ListWithBirthDate retorna membros ativos que possuem data de nascimento.
	ListWithBirthDate(ctx context.Context, workspaceID uuid.UUID) ([]FamilyMember, error)
	// UpdateAvatar grava (ou limpa, com key nil) a object key da foto do membro.
	UpdateAvatar(ctx context.Context, workspaceID, id uuid.UUID, key *string) error
}
