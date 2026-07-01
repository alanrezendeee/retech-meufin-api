package persistence

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
	"gorm.io/gorm"
)

type HealthDashboardRepository struct {
	db *gorm.DB
}

func NewHealthDashboardRepository(db *gorm.DB) *HealthDashboardRepository {
	return &HealthDashboardRepository{db: db}
}

func (r *HealthDashboardRepository) MarkerEvolution(ctx context.Context, workspaceID, markerID uuid.UUID, familyMemberID *uuid.UUID, from, to *time.Time) ([]dom.EvolutionPoint, error) {
	type row struct {
		ExamDate               time.Time
		ResultNumeric          *float64
		Unit                   *string
		ReferenceMin           *float64
		ReferenceMax           *float64
		ReferenceText          *string
		LabID                  *uuid.UUID
		Interpretation         *string
		InterpretationComputed *string
	}
	q := r.db.WithContext(ctx).
		Table("health_exam_result_items ri").
		Select("r.exam_date, ri.result_numeric, ri.unit, ri.reference_min, ri.reference_max, ri.reference_text, r.lab_id, ri.interpretation, ri.interpretation_computed").
		Joins("JOIN health_exam_results r ON r.id = ri.exam_result_id").
		Where("ri.workspace_id = ? AND ri.marker_id = ? AND ri.deleted_at IS NULL AND r.deleted_at IS NULL", workspaceID, markerID)
	if familyMemberID != nil {
		q = q.Where("r.family_member_id = ?", *familyMemberID)
	}
	if from != nil {
		q = q.Where("r.exam_date >= ?", *from)
	}
	if to != nil {
		q = q.Where("r.exam_date <= ?", *to)
	}
	q = q.Order("r.exam_date ASC")

	var rows []row
	if err := q.Scan(&rows).Error; err != nil {
		return nil, mapHealthErr(err)
	}
	out := make([]dom.EvolutionPoint, len(rows))
	for i := range rows {
		interp := rows[i].Interpretation
		if interp == nil {
			interp = rows[i].InterpretationComputed
		}
		out[i] = dom.EvolutionPoint{
			ExamDate:       rows[i].ExamDate,
			Value:          rows[i].ResultNumeric,
			Unit:           rows[i].Unit,
			RefMin:         rows[i].ReferenceMin,
			RefMax:         rows[i].ReferenceMax,
			RefText:        rows[i].ReferenceText,
			LabID:          rows[i].LabID,
			Interpretation: interp,
		}
	}
	return out, nil
}

func (r *HealthDashboardRepository) Counts(ctx context.Context, workspaceID uuid.UUID) (dom.DashboardCounts, error) {
	var c dom.DashboardCounts
	db := r.db.WithContext(ctx)
	if err := db.Model(&HealthFamilyMemberModel{}).Where("workspace_id = ?", workspaceID).Count(&c.FamilyMembers).Error; err != nil {
		return c, mapHealthErr(err)
	}
	if err := db.Model(&HealthExamResultModel{}).Where("workspace_id = ?", workspaceID).Count(&c.ExamResults).Error; err != nil {
		return c, mapHealthErr(err)
	}
	if err := db.Model(&HealthMarkerModel{}).Where("scope = ? AND workspace_id = ?", "tenant", workspaceID).Count(&c.TenantMarkers).Error; err != nil {
		return c, mapHealthErr(err)
	}
	if err := db.Table("health_documents").
		Where("workspace_id = ? AND deleted_at IS NULL AND extraction_status IN ?", workspaceID, []string{"pending", "extracted"}).
		Count(&c.DocumentsPendingReview).Error; err != nil {
		return c, mapHealthErr(err)
	}
	return c, nil
}
