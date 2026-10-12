package members

import (
	"fmt"
	"strings"
	"time"

	"sqldash/database"
	"sqldash/models"
	"sqldash/permissions"
	"sqldash/services/access"
	"sqldash/services/auth"
	"sqldash/services/mailing"
	"sqldash/utils/collections"
)

type Choice struct {
	ID       uint
	Name     string
	Color    string
	Email    string
	Initial  string
	Picture  string
	Selected bool
	Owned    bool
	Implicit bool
}

type UserView struct {
	ID        uint
	Name      string
	Username  string
	Email     string
	Initial   string
	Picture   string
	Active    bool
	Owner     bool
	Super     bool
	Manage    bool
	All       bool
	Databases int64
	Roles     []Choice
}

type UserPage struct {
	Title           string
	Users           []UserView
	Invitations     []models.AccountLink
	Roles           []Choice
	Databases       []Choice
	Editing         *UserView
	Transferring    *UserView
	Issued          *IssuedLink
	Query           string
	Page            int
	Previous        int
	Next            int
	Pages           int
	Total           int64
	Owner           bool
	CanGrantAll     bool
	RoleID          uint
	InvitationsOnly bool
	ScopeOptions    []collections.Option
	ScopeValue      string
	EmailConfigured bool
	EmailDisabled   bool
}

type PermissionView struct {
	Key         string
	Label       string
	Description string
	Selected    bool
	Available   bool
	Disabled    bool
}

type PermissionGroup struct {
	Name        string
	Permissions []PermissionView
}

type RoleView struct {
	models.Role
	Editable bool
	Members  int64
}

type RolePage struct {
	Title            string
	Roles            []RoleView
	Editing          *models.Role
	Groups           []PermissionGroup
	Editable         bool
	Creating         bool
	MaximumPosition  int
	Members          []UserView
	AvailableUsers   []UserView
	CanManageMembers bool
	MemberTab        bool
	DisplayTab       bool
	ColourOptions    []string
}

func Users(actor *access.AccountAccess, query string, page int, editing uint, issued *IssuedLink, roleFilter ...uint) (*UserPage, error) {
	if !actor.Can(permissions.UsersManage) {
		return nil, Refused
	}
	query = strings.TrimSpace(query)
	if page < 1 {
		page = 1
	}
	data := &UserPage{Title: "Users", Issued: issued, Query: query, Page: page, Owner: actor.Owner, CanGrantAll: actor.All(), ScopeValue: "false", ScopeOptions: collections.OptionsOf("false", "Selected databases")}
	mailSettings, mailError := mailing.Settings()
	if mailError != nil {
		return nil, mailError
	}
	data.EmailConfigured = mailing.Configured(mailSettings)
	data.EmailDisabled = !data.EmailConfigured
	if actor.All() {
		data.ScopeOptions = append(data.ScopeOptions, collections.Option{Value: "true", Label: "All databases"})
	}
	held := database.DB.Model(&models.Account{})
	if len(roleFilter) > 0 && roleFilter[0] != 0 {
		data.RoleID = roleFilter[0]
		held = held.Where("id IN (SELECT account_id FROM account_roles WHERE role_id = ?)", data.RoleID)
	}
	if query != "" {
		needle := "%" + query + "%"
		held = held.Where("username LIKE ? OR display_name LIKE ? OR email LIKE ?", needle, needle, needle)
	}
	if findError := held.Count(&data.Total).Error; findError != nil {
		return nil, findError
	}
	data.Pages = int((data.Total + 24) / 25)
	if data.Pages > 0 && page > data.Pages {
		page = data.Pages
		data.Page = page
	}
	data.Previous, data.Next = page-1, page+1
	var accounts []models.Account
	if findError := held.Order("id asc").Offset((page - 1) * 25).Limit(25).Find(&accounts).Error; findError != nil {
		return nil, findError
	}
	for _, record := range accounts {
		target, findError := access.Load(record.ID)
		if findError != nil {
			return nil, findError
		}
		view, viewError := userView(target)
		if viewError != nil {
			return nil, viewError
		}
		view.Manage = actor.CanManage(target)
		data.Users = append(data.Users, *view)
	}
	var roles []models.Role
	if findError := database.DB.Order("position desc").Find(&roles).Error; findError != nil {
		return nil, findError
	}
	for _, role := range roles {
		if validateGrantsWith(database.DB, actor, []uint{role.ID}, nil, false) == nil {
			data.Roles = append(data.Roles, Choice{ID: role.ID, Name: role.Name, Color: role.Color})
		}
	}
	visible, findError := access.Visible(actor)
	if findError != nil {
		return nil, findError
	}
	for _, record := range visible {
		data.Databases = append(data.Databases, Choice{ID: record.ID, Name: record.Name, Owned: record.OwnerAccountID == editing && editing != 0, Selected: record.OwnerAccountID == editing && editing != 0})
	}
	invitations := database.DB.Where("kind = ? AND used_at IS NULL AND revoked_at IS NULL AND expires_at > ?", Invitation, time.Now())
	if !actor.Owner && !actor.Super {
		invitations = invitations.Where("issuer_id = ?", actor.Account.ID)
	}
	if findError := invitations.Order("id desc").Find(&data.Invitations).Error; findError != nil {
		return nil, findError
	}
	if editing != 0 {
		target, findError := access.Load(editing)
		if findError != nil || !actor.CanManage(target) {
			return nil, Refused
		}
		view, viewError := userView(target)
		if viewError != nil {
			return nil, viewError
		}
		view.Manage = true
		data.Editing = view
		if view.All {
			data.ScopeValue = "true"
		}
		var assigned []models.AccountRole
		if findError := database.DB.Where("account_id = ?", editing).Find(&assigned).Error; findError != nil {
			return nil, findError
		}
		var shares []models.DatabaseShare
		if findError := database.DB.Where("account_id = ?", editing).Find(&shares).Error; findError != nil {
			return nil, findError
		}
		for index, choice := range data.Roles {
			for _, role := range assigned {
				if role.RoleID == choice.ID {
					data.Roles[index].Selected = true
				}
			}
		}
		for index, choice := range data.Databases {
			for _, share := range shares {
				if share.DatabaseID == choice.ID {
					data.Databases[index].Selected = true
				}
			}
		}
	}
	return data, nil
}

func userView(target *access.AccountAccess) (*UserView, error) {
	account := auth.ToAccountView(&target.Account)
	shown := &UserView{ID: target.Account.ID, Name: account.DisplayName, Username: account.Username, Email: account.Email, Initial: account.Initial, Picture: account.ProfilePicture, Active: target.Account.Active, Owner: target.Owner, Super: target.Super, All: target.All() || target.Account.AllDatabases}
	var roles []models.Role
	if findError := database.DB.Joins("JOIN account_roles ON account_roles.role_id = roles.id").Where("account_roles.account_id = ?", target.Account.ID).Order("position desc").Find(&roles).Error; findError != nil {
		return nil, findError
	}
	for _, role := range roles {
		shown.Roles = append(shown.Roles, Choice{ID: role.ID, Name: role.Name, Color: role.Color})
	}
	if countError := access.Scope(database.DB.Model(&models.Database{}), target).Count(&shown.Databases).Error; countError != nil {
		return nil, countError
	}
	return shown, nil
}

func OwnershipRecipient(actor *access.AccountAccess, identifier uint) (*UserView, error) {
	if actor == nil || !actor.Owner {
		return nil, Refused
	}
	target, findError := access.Load(identifier)
	if findError != nil || !target.Account.Active || target.Owner {
		return nil, Refused
	}
	return userView(target)
}

func Roles(actor *access.AccountAccess, editing uint, creating bool) (*RolePage, error) {
	if !actor.Can(permissions.RolesManage) {
		return nil, Refused
	}
	data := &RolePage{Title: "Roles & Permissions", Creating: creating, Editable: creating, MaximumPosition: permissions.SuperAdminPosition - 1}
	if !actor.Owner {
		data.MaximumPosition = actor.Rank - 1
	}
	var roles []models.Role
	if findError := database.DB.Order("position desc, id asc").Find(&roles).Error; findError != nil {
		return nil, findError
	}
	selected := make(map[string]bool)
	for _, role := range roles {
		view := RoleView{Role: role, Editable: actor.CanGrant(role)}
		if countError := database.DB.Model(&models.AccountRole{}).Where("role_id = ?", role.ID).Count(&view.Members).Error; countError != nil {
			return nil, countError
		}
		data.Roles = append(data.Roles, view)
		if role.ID == editing && !creating {
			copy := role
			data.Editing, data.Editable = &copy, view.Editable
			var grants []models.RolePermission
			if findError := database.DB.Where("role_id = ?", role.ID).Find(&grants).Error; findError != nil {
				return nil, findError
			}
			for _, grant := range grants {
				selected[grant.Permission] = true
			}
		}
	}
	if editing != 0 && !creating && data.Editing == nil {
		return nil, Invalid
	}
	for _, capability := range permissions.Catalogue {
		if len(data.Groups) == 0 || data.Groups[len(data.Groups)-1].Name != capability.Group {
			data.Groups = append(data.Groups, PermissionGroup{Name: capability.Group})
		}
		group := &data.Groups[len(data.Groups)-1]
		group.Permissions = append(group.Permissions, PermissionView{Key: capability.Key, Label: capability.Label, Description: capability.Description, Selected: selected[capability.Key], Available: data.Editable && actor.Can(capability.Key), Disabled: !data.Editable || !actor.Can(capability.Key)})
	}
	if data.Editing != nil && actor.Can(permissions.UsersManage) {
		data.CanManageMembers = data.Editable && ValidateGrants(actor, []uint{data.Editing.ID}, nil, false) == nil
		var findError error
		data.Members, data.AvailableUsers, findError = RoleMemberChoices(actor, data.Editing.ID)
		if findError != nil {
			return nil, findError
		}
	}
	return data, nil
}

func ParsePage(value string) int {
	var page int
	fmt.Sscan(value, &page)
	return page
}
