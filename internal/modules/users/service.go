package users

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	stderrors "errors"
	"strings"

	"gorm.io/gorm"

	apierrors "rms-be/internal/api/errors"
	"rms-be/internal/api/security"
	"rms-be/internal/middleware"
)

// Actor is the authenticated subject for authorization.
type Actor struct {
	UserID         string
	OrganizationID string
	Role           string
}

// Service contains user management business logic.
type Service struct {
	repo *Repository
	pass security.PasswordHasher
}

// NewService constructs a Service.
func NewService(repo *Repository, pass security.PasswordHasher) *Service {
	return &Service{repo: repo, pass: pass}
}

// List returns paginated users in an organization.
func (s *Service) List(ctx context.Context, actor Actor, organizationID string, page, pageSize, offset int) (*UserListResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	if organizationID == "" {
		return nil, apierrors.ErrValidation
	}
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canManageUsers(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	rows, total, err := s.repo.ListByOrganization(ctx, organizationID, offset, pageSize)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	out := make([]UserResponse, 0, len(rows))
	for i := range rows {
		resp := toResponse(&rows[i])
		if rows[i].Role == UserRoleManager {
			bids, err := s.repo.ListManagerBuildingIDs(ctx, organizationID, rows[i].ID)
			if err != nil {
				return nil, apierrors.Wrap(err, apierrors.ErrInternal)
			}
			resp.ManagerBuildingIDs = bids
		}
		out = append(out, resp)
	}
	return &UserListResponse{
		Items:    out,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}, nil
}

// Create adds a user to an organization.
func (s *Service) Create(ctx context.Context, actor Actor, organizationID string, req *CreateUserRequest) (*UserResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canManageUsers(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	assignRole, err := parseRole(req.Role)
	if err != nil {
		return nil, apierrors.ErrValidation
	}
	if !actorCanAssignRole(actor.Role, assignRole) {
		return nil, apierrors.ErrForbidden
	}
	status := parseStatus(req.Status)
	if status == "" {
		status = UserStatusActive
	}
	plain, err := resolveCreatePassword(req.Password, status)
	if err != nil {
		return nil, err
	}
	hash, err := s.pass.Hash(plain)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if taken, err := s.repo.EmailTakenByOtherUser(ctx, organizationID, email, ""); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	} else if taken {
		return nil, apierrors.ErrConflict
	}

	uid := strings.TrimSpace(actor.UserID)
	u := &User{
		OrganizationID: organizationID,
		Email:          email,
		PasswordHash:   hash,
		Role:           assignRole,
		Status:         status,
		FirstName:      strings.TrimSpace(req.FirstName),
		LastName:       strings.TrimSpace(req.LastName),
		Phone:          strings.TrimSpace(req.Phone),
	}
	if uid != "" {
		u.CreatedBy = &uid
		u.UpdatedBy = &uid
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrConflict)
	}
	assignments := normalizeStringIDs(req.ManagerBuildingIDs)
	if u.Role == UserRoleManager {
		ok, err := s.validateManagerBuildingAssignments(ctx, organizationID, assignments)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, apierrors.ErrValidation
		}
		if err := s.repo.ReplaceManagerBuildingAssignments(ctx, organizationID, u.ID, assignments); err != nil {
			return nil, apierrors.Wrap(err, apierrors.ErrInternal)
		}
	}
	r := toResponse(u)
	if u.Role == UserRoleManager {
		r.ManagerBuildingIDs = assignments
	}
	return &r, nil
}

// Patch updates a user in an organization.
func (s *Service) Patch(ctx context.Context, actor Actor, organizationID, userID string, req *PatchUserRequest) (*UserResponse, error) {
	organizationID = strings.TrimSpace(organizationID)
	userID = strings.TrimSpace(userID)
	if err := assertOrgScope(actor, organizationID); err != nil {
		return nil, err
	}
	if !canManageUsers(actor.Role) {
		return nil, apierrors.ErrForbidden
	}
	target, err := s.repo.GetByIDInOrg(ctx, organizationID, userID)
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.ErrNotFound
		}
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if !actorCanModifyTarget(actor.Role, target) {
		return nil, apierrors.ErrForbidden
	}

	updates := make(map[string]any)
	if req.Email != nil {
		email := strings.TrimSpace(strings.ToLower(*req.Email))
		if email == "" {
			return nil, apierrors.ErrValidation
		}
		taken, err := s.repo.EmailTakenByOtherUser(ctx, organizationID, email, userID)
		if err != nil {
			return nil, apierrors.Wrap(err, apierrors.ErrInternal)
		}
		if taken {
			return nil, apierrors.ErrConflict
		}
		updates["email"] = email
	}
	if req.Password != nil {
		p := strings.TrimSpace(*req.Password)
		if len(p) < 8 {
			return nil, apierrors.ErrValidation
		}
		hash, err := s.pass.Hash(p)
		if err != nil {
			return nil, apierrors.Wrap(err, apierrors.ErrInternal)
		}
		updates["password_hash"] = hash
	}
	if req.FirstName != nil {
		v := strings.TrimSpace(*req.FirstName)
		if v == "" {
			return nil, apierrors.ErrValidation
		}
		updates["first_name"] = v
	}
	if req.LastName != nil {
		updates["last_name"] = strings.TrimSpace(*req.LastName)
	}
	if req.Phone != nil {
		updates["phone"] = strings.TrimSpace(*req.Phone)
	}
	if req.Role != nil {
		newRole, err := parseRole(*req.Role)
		if err != nil {
			return nil, apierrors.ErrValidation
		}
		if !actorCanAssignRole(actor.Role, newRole) {
			return nil, apierrors.ErrForbidden
		}
		// Simulate post-update target for policy: same org user with new role
		twin := *target
		twin.Role = newRole
		if !actorCanModifyTarget(actor.Role, &twin) {
			return nil, apierrors.ErrForbidden
		}
		updates["role"] = string(newRole)
	}
	assignments := []string(nil)
	if req.ManagerBuildingIDs != nil {
		assignments = normalizeStringIDs(*req.ManagerBuildingIDs)
	}
	if req.Status != nil {
		st := parseStatus(*req.Status)
		if st == "" {
			return nil, apierrors.ErrValidation
		}
		updates["status"] = string(st)
	}
	if len(updates) == 0 {
		r := toResponse(target)
		return &r, nil
	}
	uid := strings.TrimSpace(actor.UserID)
	if uid != "" {
		updates["updated_by"] = uid
	}
	if err := s.repo.Update(ctx, organizationID, userID, updates); err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	refreshed, err := s.repo.GetByIDInOrg(ctx, organizationID, userID)
	if err != nil {
		return nil, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	if req.ManagerBuildingIDs != nil || refreshed.Role == UserRoleManager {
		switch refreshed.Role {
		case UserRoleManager:
			ok, err := s.validateManagerBuildingAssignments(ctx, organizationID, assignments)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, apierrors.ErrValidation
			}
			if err := s.repo.ReplaceManagerBuildingAssignments(ctx, organizationID, refreshed.ID, assignments); err != nil {
				return nil, apierrors.Wrap(err, apierrors.ErrInternal)
			}
		default:
			if err := s.repo.ReplaceManagerBuildingAssignments(ctx, organizationID, refreshed.ID, nil); err != nil {
				return nil, apierrors.Wrap(err, apierrors.ErrInternal)
			}
		}
	}
	r := toResponse(refreshed)
	if refreshed.Role == UserRoleManager {
		bids, err := s.repo.ListManagerBuildingIDs(ctx, organizationID, refreshed.ID)
		if err != nil {
			return nil, apierrors.Wrap(err, apierrors.ErrInternal)
		}
		r.ManagerBuildingIDs = bids
	}
	return &r, nil
}

func resolveCreatePassword(password string, status UserStatus) (string, error) {
	p := strings.TrimSpace(password)
	switch status {
	case UserStatusInvited:
		if p != "" {
			return p, nil
		}
		return randomPlainPassword()
	default:
		if p == "" {
			return "", apierrors.ErrValidation
		}
		return p, nil
	}
}

func randomPlainPassword() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", apierrors.Wrap(err, apierrors.ErrInternal)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func assertOrgScope(actor Actor, organizationID string) error {
	if strings.EqualFold(strings.TrimSpace(actor.Role), middleware.RoleAdmin) {
		return nil
	}
	if !strings.EqualFold(strings.TrimSpace(actor.OrganizationID), organizationID) {
		return apierrors.ErrForbidden
	}
	return nil
}

func canManageUsers(role string) bool {
	r := strings.TrimSpace(strings.ToLower(role))
	switch r {
	case middleware.RoleAdmin, middleware.RoleLandlord, middleware.RoleManager:
		return true
	default:
		return false
	}
}

func actorCanAssignRole(actorRole string, assign UserRole) bool {
	a := strings.TrimSpace(strings.ToLower(actorRole))
	switch a {
	case middleware.RoleAdmin:
		return true
	case middleware.RoleLandlord:
		return assign == UserRoleManager || assign == UserRoleStaff || assign == UserRoleTenant
	case middleware.RoleManager:
		return assign == UserRoleStaff || assign == UserRoleTenant
	default:
		return false
	}
}

func actorCanModifyTarget(actorRole string, target *User) bool {
	a := strings.TrimSpace(strings.ToLower(actorRole))
	switch a {
	case middleware.RoleAdmin:
		return true
	case middleware.RoleLandlord:
		return target.Role == UserRoleManager || target.Role == UserRoleStaff || target.Role == UserRoleTenant
	case middleware.RoleManager:
		return target.Role == UserRoleStaff || target.Role == UserRoleTenant
	default:
		return false
	}
}

func parseRole(s string) (UserRole, error) {
	r := UserRole(strings.TrimSpace(strings.ToLower(s)))
	switch r {
	case UserRoleAdmin, UserRoleLandlord, UserRoleManager, UserRoleStaff, UserRoleTenant:
		return r, nil
	default:
		var z UserRole
		return z, apierrors.ErrValidation
	}
}

func parseStatus(s string) UserStatus {
	st := UserStatus(strings.TrimSpace(strings.ToLower(s)))
	switch st {
	case UserStatusActive, UserStatusDisabled, UserStatusInvited:
		return st
	default:
		return ""
	}
}

func toResponse(u *User) UserResponse {
	return UserResponse{
		ID:             u.ID,
		OrganizationID: u.OrganizationID,
		Email:          u.Email,
		Role:           string(u.Role),
		Status:         string(u.Status),
		FirstName:      u.FirstName,
		LastName:       u.LastName,
		Phone:          u.Phone,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
		CreatedBy:      u.CreatedBy,
		UpdatedBy:      u.UpdatedBy,
	}
}

func normalizeStringIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	seen := make(map[string]bool, len(ids))
	for _, raw := range ids {
		v := strings.TrimSpace(raw)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func (s *Service) validateManagerBuildingAssignments(ctx context.Context, organizationID string, ids []string) (bool, error) {
	if len(ids) == 0 {
		return true, nil
	}
	n, err := s.repo.CountBuildingsInOrg(ctx, organizationID, ids)
	if err != nil {
		return false, apierrors.Wrap(err, apierrors.ErrInternal)
	}
	return int64(len(ids)) == n, nil
}
