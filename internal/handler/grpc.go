package handler

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	splixv1 "github.com/halooid/backend/splix-service/gen/go/splix/v1"
	"github.com/halooid/backend/splix-service/internal/db"
	"github.com/halooid/backend/splix-service/internal/service"
	"github.com/halooid/backend/go-shared/auth"
	"google.golang.org/protobuf/types/known/timestamppb"
)


type SplixHandler struct {
	splixv1.UnimplementedUserServiceServer
	splixv1.UnimplementedGroupServiceServer
	splixv1.UnimplementedExpenseServiceServer
	service *service.SplixService
}

func NewSplixHandler(service *service.SplixService) *SplixHandler {
	return &SplixHandler{service: service}
}

// User Service
func (h *SplixHandler) CreateUser(ctx context.Context, req *splixv1.CreateUserRequest) (*splixv1.CreateUserResponse, error) {
	id, _ := uuid.Parse(req.Id)
	u, err := h.service.CreateUser(ctx, id, req.Email, req.Name, req.CountryCode, req.DefaultCurrencyCode)
	if err != nil {
		return nil, err
	}
	return &splixv1.CreateUserResponse{User: mapUser(u)}, nil
}

func (h *SplixHandler) GetUser(ctx context.Context, req *splixv1.GetUserRequest) (*splixv1.GetUserResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, fmt.Errorf("invalid user id: %w", err)
	}
	u, err := h.service.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}
	return &splixv1.GetUserResponse{User: mapUser(u)}, nil
}

func (h *SplixHandler) AddConnection(ctx context.Context, req *splixv1.AddConnectionRequest) (*splixv1.AddConnectionResponse, error) {
	claims, ok := auth.GetUser(ctx)
	if !ok {
		return nil, fmt.Errorf("unauthenticated")
	}
	userID, _ := uuid.Parse(claims.Subject)
	connectedID, _ := uuid.Parse(req.ConnectedUserId)
	err := h.service.AddConnection(ctx, userID, connectedID)
	if err != nil {
		return nil, err
	}
	return &splixv1.AddConnectionResponse{Success: true}, nil
}

func (h *SplixHandler) ListConnections(ctx context.Context, req *splixv1.ListConnectionsRequest) (*splixv1.ListConnectionsResponse, error) {
	claims, ok := auth.GetUser(ctx)
	if !ok {
		return nil, fmt.Errorf("unauthenticated")
	}
	userID, _ := uuid.Parse(claims.Subject)
	users, err := h.service.ListConnections(ctx, userID)
	if err != nil {
		return nil, err
	}
	protoUsers := make([]*splixv1.SplixUser, len(users))
	for i, u := range users {
		protoUsers[i] = mapUser(u)
	}
	return &splixv1.ListConnectionsResponse{Connections: protoUsers}, nil
}

// Group Service
func (h *SplixHandler) CreateGroup(ctx context.Context, req *splixv1.CreateGroupRequest) (*splixv1.CreateGroupResponse, error) {
	claims, ok := auth.GetUser(ctx)
	if !ok {
		return nil, fmt.Errorf("unauthenticated")
	}
	userID, _ := uuid.Parse(claims.Subject)
	memberIDs := make([]uuid.UUID, len(req.MemberIds))
	for i, m := range req.MemberIds {
		memberIDs[i], _ = uuid.Parse(m)
	}

	g, err := h.service.CreateGroup(ctx, req.Name, userID, memberIDs)
	if err != nil {
		return nil, err
	}
	return &splixv1.CreateGroupResponse{Group: mapGroup(g)}, nil
}

func (h *SplixHandler) AddGroupMember(ctx context.Context, req *splixv1.AddGroupMemberRequest) (*splixv1.AddGroupMemberResponse, error) {
	groupID, _ := uuid.Parse(req.GroupId)
	userID, _ := uuid.Parse(req.UserId)
	err := h.service.AddGroupMember(ctx, groupID, userID)
	if err != nil {
		return nil, err
	}
	return &splixv1.AddGroupMemberResponse{Success: true}, nil
}

func (h *SplixHandler) ListGroups(ctx context.Context, req *splixv1.ListGroupsRequest) (*splixv1.ListGroupsResponse, error) {
	claims, ok := auth.GetUser(ctx)
	if !ok {
		return nil, fmt.Errorf("unauthenticated")
	}
	userID, _ := uuid.Parse(claims.Subject)
	groups, err := h.service.ListGroups(ctx, userID)
	if err != nil {
		return nil, err
	}
	protoGroups := make([]*splixv1.Group, len(groups))
	for i, g := range groups {
		protoGroups[i] = mapGroup(g)
	}
	return &splixv1.ListGroupsResponse{Groups: protoGroups}, nil
}

// Expense Service
func (h *SplixHandler) AddExpense(ctx context.Context, req *splixv1.AddExpenseRequest) (*splixv1.AddExpenseResponse, error) {
	claims, ok := auth.GetUser(ctx)
	if !ok {
		return nil, fmt.Errorf("unauthenticated")
	}
	userID, _ := uuid.Parse(claims.Subject)

	var groupID *uuid.UUID
	if req.GroupId != "" {
		id, _ := uuid.Parse(req.GroupId)
		groupID = &id
	}

	splits := make([]service.Split, len(req.Splits))
	for i, s := range req.Splits {
		uID, _ := uuid.Parse(s.UserId)
		splits[i] = service.Split{
			UserID:     uID,
			PaidAmount: s.PaidAmount,
			OwedAmount: s.OwedAmount,
		}
	}

	e, sSplits, err := h.service.AddExpense(ctx, groupID, req.Description, req.CurrencyCode, req.TotalAmount, userID, splits)
	if err != nil {
		return nil, err
	}

	return &splixv1.AddExpenseResponse{Expense: mapExpense(e, sSplits)}, nil
}

func (h *SplixHandler) GetUserBalances(ctx context.Context, req *splixv1.GetUserBalancesRequest) (*splixv1.GetUserBalancesResponse, error) {
	claims, ok := auth.GetUser(ctx)
	if !ok {
		return nil, fmt.Errorf("unauthenticated")
	}
	userID, _ := uuid.Parse(claims.Subject)
	balances, err := h.service.GetUserBalances(ctx, userID)
	if err != nil {
		return nil, err
	}
	protoBalances := make([]*splixv1.UserBalance, len(balances))
	for i, b := range balances {
		bal, _ := strconv.ParseFloat(fmt.Sprintf("%v", b.Balance), 64)
		protoBalances[i] = &splixv1.UserBalance{
			UserId:       userID.String(),
			CurrencyCode: b.CurrencyCode,
			Balance:      bal,
		}
	}
	return &splixv1.GetUserBalancesResponse{Balances: protoBalances}, nil
}

func (h *SplixHandler) GetUserExpenses(ctx context.Context, req *splixv1.GetUserExpensesRequest) (*splixv1.GetUserExpensesResponse, error) {
	claims, ok := auth.GetUser(ctx)
	if !ok {
		return nil, fmt.Errorf("unauthenticated")
	}
	userID, _ := uuid.Parse(claims.Subject)
	var groupID *uuid.UUID
	if req.GroupId != "" {
		id, _ := uuid.Parse(req.GroupId)
		groupID = &id
	}

	expenses, err := h.service.GetUserExpenses(ctx, userID, groupID)
	if err != nil {
		return nil, err
	}

	protoExpenses := make([]*splixv1.Expense, len(expenses))
	for i, e := range expenses {
		splits, _ := h.service.GetExpenseSplits(ctx, e.ID)
		protoExpenses[i] = mapExpense(e, splits)
	}
	return &splixv1.GetUserExpensesResponse{Expenses: protoExpenses}, nil
}

// Helpers
func mapUser(u db.User) *splixv1.SplixUser {
	return &splixv1.SplixUser{
		Id:                  u.ID.String(),
		Email:               u.Email,
		Name:                u.Name,
		CountryCode:         u.CountryCode.String,
		DefaultCurrencyCode: u.DefaultCurrencyCode.String,
		CreatedAt:           timestamppb.New(u.CreatedAt.Time),
	}
}

func mapGroup(g db.Group) *splixv1.Group {
	return &splixv1.Group{
		Id:        g.ID.String(),
		Name:      g.Name,
		CreatedBy: g.CreatedBy.String(),
		CreatedAt: timestamppb.New(g.CreatedAt.Time),
	}
}

func mapExpense(e db.Expense, splits []db.ExpenseSplit) *splixv1.Expense {
	protoSplits := make([]*splixv1.ExpenseSplit, len(splits))
	for i, s := range splits {
		paid, _ := strconv.ParseFloat(fmt.Sprintf("%v", s.PaidAmount), 64)
		owed, _ := strconv.ParseFloat(fmt.Sprintf("%v", s.OwedAmount), 64)
		protoSplits[i] = &splixv1.ExpenseSplit{
			UserId:     s.UserID.String(),
			PaidAmount: paid,
			OwedAmount: owed,
		}
	}

	total, _ := strconv.ParseFloat(fmt.Sprintf("%v", e.TotalAmount), 64)
	return &splixv1.Expense{
		Id:           e.ID.String(),
		GroupId:      e.GroupID.UUID.String(),
		Description:  e.Description,
		CurrencyCode: e.CurrencyCode,
		TotalAmount:  total,
		CreatedBy:    e.CreatedBy.String(),
		CreatedAt:    timestamppb.New(e.CreatedAt.Time),
		Splits:       protoSplits,
	}
}
