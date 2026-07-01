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
	Active       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
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
	return nil
}

// FamilyMemberRepository abstrai a persistência de membros da família (workspace-scoped).
type FamilyMemberRepository interface {
	Create(ctx context.Context, f *FamilyMember) error
	GetByID(ctx context.Context, workspaceID, id uuid.UUID) (*FamilyMember, error)
	Update(ctx context.Context, f *FamilyMember) error
	SoftDelete(ctx context.Context, workspaceID, id uuid.UUID) error
	List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) ([]FamilyMember, int64, error)
}
