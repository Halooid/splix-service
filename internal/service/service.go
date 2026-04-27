package service

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/halooid/backend/splix-service/internal/db"
)

type SplixService struct {
	queries db.Querier
}

func NewSplixService(queries db.Querier) *SplixService {
	return &SplixService{queries: queries}
}

// User Logic
func (s *SplixService) CreateUser(ctx context.Context, id uuid.UUID, email, name, countryCode, currencyCode string) (db.User, error) {
	return s.queries.CreateUser(ctx, db.CreateUserParams{
		ID:                  id,
		Email:               email,
		Name:                name,
		CountryCode:         sql.NullString{String: countryCode, Valid: countryCode != ""},
		DefaultCurrencyCode: sql.NullString{String: currencyCode, Valid: currencyCode != ""},
	})
}

func (s *SplixService) AddConnection(ctx context.Context, userID, connectedUserID uuid.UUID) error {
	return s.queries.AddConnection(ctx, db.AddConnectionParams{
		UserID:          userID,
		ConnectedUserID: connectedUserID,
	})
}

func (s *SplixService) ListConnections(ctx context.Context, userID uuid.UUID) ([]db.User, error) {
	return s.queries.ListConnections(ctx, userID)
}

// Group Logic
func (s *SplixService) CreateGroup(ctx context.Context, name string, createdBy uuid.UUID, memberIDs []uuid.UUID) (db.Group, error) {
	group, err := s.queries.CreateGroup(ctx, db.CreateGroupParams{
		Name:      name,
		CreatedBy: createdBy,
	})
	if err != nil {
		return db.Group{}, err
	}

	// Add members
	for _, mID := range memberIDs {
		_ = s.queries.AddGroupMember(ctx, db.AddGroupMemberParams{
			GroupID: group.ID,
			UserID:  mID,
		})
	}
	// Also add the creator as a member if not already in the list
	_ = s.queries.AddGroupMember(ctx, db.AddGroupMemberParams{
		GroupID: group.ID,
		UserID:  createdBy,
	})

	return group, nil
}

func (s *SplixService) AddGroupMember(ctx context.Context, groupID, userID uuid.UUID) error {
	return s.queries.AddGroupMember(ctx, db.AddGroupMemberParams{
		GroupID: groupID,
		UserID:  userID,
	})
}

func (s *SplixService) ListGroups(ctx context.Context, userID uuid.UUID) ([]db.Group, error) {
	return s.queries.ListGroups(ctx, userID)
}

func (s *SplixService) GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	return s.queries.GetGroupMembers(ctx, groupID)
}

// Expense Logic
type Split struct {
	UserID     uuid.UUID
	PaidAmount float64
	OwedAmount float64
}

func (s *SplixService) AddExpense(ctx context.Context, groupID *uuid.UUID, description, currencyCode string, totalAmount float64, createdBy uuid.UUID, splits []Split) (db.Expense, []db.ExpenseSplit, error) {
	expense, err := s.queries.CreateExpense(ctx, db.CreateExpenseParams{
		GroupID:      uuidToNullUUID(groupID),
		Description:  description,
		CurrencyCode: currencyCode,
		TotalAmount:  fmtNumeric(totalAmount),
		CreatedBy:    createdBy,
	})
	if err != nil {
		return db.Expense{}, nil, err
	}

	var dbSplits []db.ExpenseSplit
	for _, split := range splits {
		err := s.queries.CreateExpenseSplit(ctx, db.CreateExpenseSplitParams{
			ExpenseID:  expense.ID,
			UserID:     split.UserID,
			PaidAmount: fmtNumeric(split.PaidAmount),
			OwedAmount: fmtNumeric(split.OwedAmount),
		})
		if err == nil {
			dbSplits = append(dbSplits, db.ExpenseSplit{
				ExpenseID:  expense.ID,
				UserID:     split.UserID,
				PaidAmount: fmtNumeric(split.PaidAmount),
				OwedAmount: fmtNumeric(split.OwedAmount),
			})
		}
	}

	return expense, dbSplits, nil
}

func (s *SplixService) GetUserBalances(ctx context.Context, userID uuid.UUID) ([]db.GetUserBalancesRow, error) {
	return s.queries.GetUserBalances(ctx, userID)
}

func (s *SplixService) GetUserExpenses(ctx context.Context, userID uuid.UUID, groupID *uuid.UUID) ([]db.Expense, error) {
	return s.queries.GetUserExpenses(ctx, db.GetUserExpensesParams{
		UserID:  userID,
		GroupID: uuidToNullUUID(groupID),
	})
}

func (s *SplixService) GetExpenseSplits(ctx context.Context, expenseID uuid.UUID) ([]db.ExpenseSplit, error) {
	return s.queries.GetExpenseSplits(ctx, expenseID)
}

// Helpers
func uuidToNullUUID(id *uuid.UUID) uuid.NullUUID {
	if id == nil {
		return uuid.NullUUID{Valid: false}
	}
	return uuid.NullUUID{UUID: *id, Valid: true}
}

func fmtNumeric(f float64) string {
	return fmt.Sprintf("%.2f", f)
}
