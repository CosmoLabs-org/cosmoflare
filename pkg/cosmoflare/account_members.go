package cosmoflare

import (
	"context"
	"fmt"

	"github.com/cloudflare/cloudflare-go"
)

// AccountMemberRoleView is a role attached to an account member (or, from
// RolesList, a role available for assignment in the account).
type AccountMemberRoleView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// AccountMemberView is the public view of an account member. Email is the
// canonical identifier for invites; ID is the canonical identifier for
// update/remove.
type AccountMemberView struct {
	ID        string                  `json:"id"`
	Email     string                  `json:"email"`
	Name      string                  `json:"name,omitempty"`
	Status    string                  `json:"status"`
	TwoFactor bool                    `json:"twoFactorEnabled"`
	Roles     []AccountMemberRoleView `json:"roles"`
}

// AccountMemberService manages account members and roles over cloudflare-go
// (the local mutation audit log in cmd/audit.go is a separate feature; this
// is Cloudflare account membership as seen by the API).
type AccountMemberService struct {
	cf        *cloudflare.API
	accountID string
}

// NewAccountMemberService creates a member/role service from an existing
// cloudflare-go client.
func NewAccountMemberService(api *cloudflare.API, accountID string) (*AccountMemberService, error) {
	if api == nil {
		return nil, validationError("NewAccountMemberService", "cloudflare API client is required")
	}
	if accountID == "" {
		return nil, validationError("NewAccountMemberService", "account ID is required")
	}
	return &AccountMemberService{cf: api, accountID: accountID}, nil
}

// NewAccountMemberServiceFromCreds creates a member/role service from
// account ID and API token credentials.
func NewAccountMemberServiceFromCreds(accountID, apiToken string) (*AccountMemberService, error) {
	if accountID == "" {
		return nil, validationError("NewAccountMemberService", "account ID is required")
	}
	if apiToken == "" {
		return nil, validationError("NewAccountMemberService", "API token is required")
	}
	cf, err := newCloudflareAPI(apiToken)
	if err != nil {
		return nil, authError("NewAccountMemberService", "failed to create Cloudflare API client", err)
	}
	return &AccountMemberService{cf: cf, accountID: accountID}, nil
}

// List returns every member of the account, following the API's pagination.
func (s *AccountMemberService) List(ctx context.Context) ([]AccountMemberView, error) {
	const op = "AccountMemberList"
	var out []AccountMemberView
	page := 1
	for {
		members, info, err := s.cf.AccountMembers(ctx, s.accountID, cloudflare.PaginationOptions{Page: page, PerPage: 50})
		if err != nil {
			return nil, newError(op, fmt.Sprintf("failed to list members of account %q", s.accountID), err)
		}
		for _, m := range members {
			out = append(out, accountMemberViewFromCF(m))
		}
		if page >= info.TotalPages || len(members) == 0 {
			break
		}
		page++
	}
	if out == nil {
		out = []AccountMemberView{}
	}
	return out, nil
}

// Invite sends a membership invitation to an email address with the given
// role IDs. The invitee receives an email confirmation; the member starts in
// "pending" status.
func (s *AccountMemberService) Invite(ctx context.Context, email string, roleIDs []string) (*AccountMemberView, error) {
	const op = "AccountMemberInvite"
	if email == "" {
		return nil, validationError(op, "email is required")
	}
	if len(roleIDs) == 0 {
		return nil, validationError(op, "at least one role ID is required (see 'cosmoflare account role list')")
	}

	member, err := s.cf.CreateAccountMember(ctx, cloudflare.AccountIdentifier(s.accountID), cloudflare.CreateAccountMemberParams{
		EmailAddress: email,
		Roles:        roleIDs,
	})
	if err != nil {
		return nil, newError(op, fmt.Sprintf("failed to invite %q to account %q", email, s.accountID), err)
	}
	view := accountMemberViewFromCF(member)
	return &view, nil
}

// UpdateRoles replaces the set of roles held by an account member. The role
// list is the full replacement set, not a delta.
func (s *AccountMemberService) UpdateRoles(ctx context.Context, memberID string, roleIDs []string) (*AccountMemberView, error) {
	const op = "AccountMemberUpdateRoles"
	if memberID == "" {
		return nil, validationError(op, "member ID is required")
	}
	if len(roleIDs) == 0 {
		return nil, validationError(op, "at least one role ID is required (see 'cosmoflare account role list')")
	}

	roles := make([]cloudflare.AccountRole, 0, len(roleIDs))
	for _, id := range roleIDs {
		roles = append(roles, cloudflare.AccountRole{ID: id})
	}

	member, err := s.cf.UpdateAccountMember(ctx, s.accountID, memberID, cloudflare.AccountMember{Roles: roles})
	if err != nil {
		return nil, newError(op, fmt.Sprintf("failed to update roles of member %q", memberID), err)
	}
	view := accountMemberViewFromCF(member)
	return &view, nil
}

// Remove removes a member from the account. The member loses all access
// immediately; pending invitations are revoked.
func (s *AccountMemberService) Remove(ctx context.Context, memberID string) error {
	const op = "AccountMemberRemove"
	if memberID == "" {
		return validationError(op, "member ID is required")
	}
	if err := s.cf.DeleteAccountMember(ctx, s.accountID, memberID); err != nil {
		return newError(op, fmt.Sprintf("failed to remove member %q", memberID), err)
	}
	return nil
}

// RolesList returns the roles available for assignment in the account.
func (s *AccountMemberService) RolesList(ctx context.Context) ([]AccountMemberRoleView, error) {
	const op = "AccountRolesList"
	roles, err := s.cf.ListAccountRoles(ctx, cloudflare.AccountIdentifier(s.accountID), cloudflare.ListAccountRolesParams{})
	if err != nil {
		return nil, newError(op, fmt.Sprintf("failed to list roles of account %q", s.accountID), err)
	}
	out := make([]AccountMemberRoleView, 0, len(roles))
	for _, r := range roles {
		out = append(out, AccountMemberRoleView{ID: r.ID, Name: r.Name, Description: r.Description})
	}
	return out, nil
}

// accountMemberViewFromCF maps the cloudflare-go member type to the public
// view, collapsing the user details struct into flat fields.
func accountMemberViewFromCF(m cloudflare.AccountMember) AccountMemberView {
	name := fmt.Sprintf("%s %s", m.User.FirstName, m.User.LastName)
	view := AccountMemberView{
		ID:        m.ID,
		Email:     m.User.Email,
		Status:    m.Status,
		TwoFactor: m.User.TwoFactorAuthenticationEnabled,
		Roles:     make([]AccountMemberRoleView, 0, len(m.Roles)),
	}
	if name != " " {
		view.Name = name
	}
	for _, r := range m.Roles {
		view.Roles = append(view.Roles, AccountMemberRoleView{ID: r.ID, Name: r.Name, Description: r.Description})
	}
	return view
}
