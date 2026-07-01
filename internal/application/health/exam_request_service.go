package health

import (
	"context"
	"time"

	"github.com/google/uuid"
	dom "github.com/retechfin/retechfin-api/internal/domain/health"
)

type ExamRequestService struct {
	repo dom.ExamRequestRepository
}

func NewExamRequestService(repo dom.ExamRequestRepository) *ExamRequestService {
	return &ExamRequestService{repo: repo}
}

type CreateExamRequestItemInput struct {
	MarkerID *uuid.UUID
	ExamName string
	ExamCode *string
	BodyArea *string
	Notes    *string
	Status   dom.ExamRequestItemStatus
}

type CreateExamRequestInput struct {
	WorkspaceID    uuid.UUID
	FamilyMemberID uuid.UUID
	LabID          *uuid.UUID
	RequestedBy    *string
	RequestDate    *time.Time
	Status         dom.ExamRequestStatus
	Notes          *string
	Items          []CreateExamRequestItemInput
}

type UpdateExamRequestInput struct {
	WorkspaceID    uuid.UUID
	ID             uuid.UUID
	FamilyMemberID uuid.UUID
	LabID          *uuid.UUID
	RequestedBy    *string
	RequestDate    *time.Time
	Status         dom.ExamRequestStatus
	Notes          *string
}

type AddExamRequestItemInput struct {
	WorkspaceID   uuid.UUID
	ExamRequestID uuid.UUID
	MarkerID      *uuid.UUID
	ExamName      string
	ExamCode      *string
	BodyArea      *string
	Notes         *string
	Status        dom.ExamRequestItemStatus
}

type UpdateExamRequestItemInput struct {
	WorkspaceID   uuid.UUID
	ExamRequestID uuid.UUID
	ItemID        uuid.UUID
	MarkerID      *uuid.UUID
	ExamName      string
	ExamCode      *string
	BodyArea      *string
	Notes         *string
	Status        dom.ExamRequestItemStatus
}

func (s *ExamRequestService) Create(ctx context.Context, in CreateExamRequestInput) (*dom.ExamRequest, error) {
	now := time.Now().UTC()

	reqDate := now
	if in.RequestDate != nil {
		reqDate = in.RequestDate.UTC()
	}

	req := &dom.ExamRequest{
		ID:             uuid.New(),
		WorkspaceID:    in.WorkspaceID,
		FamilyMemberID: in.FamilyMemberID,
		LabID:          in.LabID,
		RequestedBy:    in.RequestedBy,
		RequestDate:    reqDate,
		Status:         in.Status,
		Notes:          in.Notes,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	for _, itin := range in.Items {
		req.Items = append(req.Items, dom.ExamRequestItem{
			ID:            uuid.New(),
			WorkspaceID:   in.WorkspaceID,
			ExamRequestID: req.ID,
			MarkerID:      itin.MarkerID,
			ExamName:      itin.ExamName,
			ExamCode:      itin.ExamCode,
			BodyArea:      itin.BodyArea,
			Notes:         itin.Notes,
			Status:        itin.Status,
			CreatedAt:     now,
			UpdatedAt:     now,
		})
	}

	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

func (s *ExamRequestService) Get(ctx context.Context, workspaceID, id uuid.UUID) (*dom.ExamRequest, error) {
	return s.repo.GetByID(ctx, workspaceID, id)
}

type ListExamRequestsResult struct {
	Items []dom.ExamRequest
	Total int64
}

func (s *ExamRequestService) List(ctx context.Context, workspaceID uuid.UUID, limit, offset int) (*ListExamRequestsResult, error) {
	items, total, err := s.repo.List(ctx, workspaceID, limit, offset)
	if err != nil {
		return nil, err
	}
	return &ListExamRequestsResult{Items: items, Total: total}, nil
}

func (s *ExamRequestService) Update(ctx context.Context, in UpdateExamRequestInput) (*dom.ExamRequest, error) {
	req, err := s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
	if err != nil {
		return nil, err
	}

	req.FamilyMemberID = in.FamilyMemberID
	req.LabID = in.LabID
	req.RequestedBy = in.RequestedBy
	if in.RequestDate != nil {
		req.RequestDate = in.RequestDate.UTC()
	}
	req.Status = in.Status
	req.Notes = in.Notes
	req.UpdatedAt = time.Now().UTC()

	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, req); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, in.WorkspaceID, in.ID)
}

func (s *ExamRequestService) Delete(ctx context.Context, workspaceID, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, workspaceID, id)
}

func (s *ExamRequestService) AddItem(ctx context.Context, in AddExamRequestItemInput) (*dom.ExamRequestItem, error) {
	now := time.Now().UTC()
	it := &dom.ExamRequestItem{
		ID:            uuid.New(),
		WorkspaceID:   in.WorkspaceID,
		ExamRequestID: in.ExamRequestID,
		MarkerID:      in.MarkerID,
		ExamName:      in.ExamName,
		ExamCode:      in.ExamCode,
		BodyArea:      in.BodyArea,
		Notes:         in.Notes,
		Status:        in.Status,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := it.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.AddItem(ctx, it); err != nil {
		return nil, err
	}
	return it, nil
}

func (s *ExamRequestService) UpdateItem(ctx context.Context, in UpdateExamRequestItemInput) (*dom.ExamRequestItem, error) {
	it := &dom.ExamRequestItem{
		ID:            in.ItemID,
		WorkspaceID:   in.WorkspaceID,
		ExamRequestID: in.ExamRequestID,
		MarkerID:      in.MarkerID,
		ExamName:      in.ExamName,
		ExamCode:      in.ExamCode,
		BodyArea:      in.BodyArea,
		Notes:         in.Notes,
		Status:        in.Status,
		UpdatedAt:     time.Now().UTC(),
	}
	if err := it.Validate(); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateItem(ctx, it); err != nil {
		return nil, err
	}
	return it, nil
}

func (s *ExamRequestService) DeleteItem(ctx context.Context, workspaceID, requestID, itemID uuid.UUID) error {
	return s.repo.SoftDeleteItem(ctx, workspaceID, requestID, itemID)
}
