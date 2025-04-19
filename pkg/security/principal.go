package security

type Principal interface {
	ComputePrivileges(hasPermissionsList ...HasPermissions) Privileges
}

type principal struct {
	user   string
	groups []string
}

func (p *principal) User() string {
	return p.user
}

func (p *principal) Groups() []string {
	return p.groups
}

func NewPrincipal(user User) Principal {
	var groups []string

	for _, group := range user.Groups {
		groups = append(groups, group.Name)
	}

	return &principal{
		user:   user.Name,
		groups: groups,
	}
}

func (p principal) ComputePrivileges(hasPermissionList ...HasPermissions) Privileges {
	canRead := false
	canWrite := false
	canDelete := false

	for _, hasPermissions := range hasPermissionList {
		permissions := hasPermissions.GetPermissions()

		if permissions.OtherPermission.Read {
			canRead = true
		}

		if permissions.OtherPermission.Write {
			canWrite = true
		}

		if permissions.OtherPermission.Delete {
			canDelete = true
		}

		if permissions.Owner == p.user {
			if permissions.OwnerPermission.Read {
				canRead = true
			}

			if permissions.OwnerPermission.Write {
				canWrite = true
			}

			if permissions.OwnerPermission.Delete {
				canDelete = true
			}
		}

		for _, group := range p.groups {
			if permissions.Group == group {
				if permissions.GroupPermission.Read {
					canRead = true
				}

				if permissions.GroupPermission.Write {
					canWrite = true
				}

				if permissions.GroupPermission.Delete {
					canDelete = true
				}

				if canRead && canWrite && canDelete {
					return Privileges{
						Read:   true,
						Write:  true,
						Delete: true,
					}
				}
			}
		}

		if canRead && canWrite && canDelete {
			return Privileges{
				Read:   true,
				Write:  true,
				Delete: true,
			}
		}
	}

	return Privileges{
		Read:   canRead,
		Write:  canWrite,
		Delete: canDelete,
	}
}

type rootPrincipal struct {
}

func NewRootPrincipal() Principal {
	return &rootPrincipal{}
}

func (p rootPrincipal) ComputePrivileges(_ ...HasPermissions) Privileges {
	return Privileges{
		Read:   true,
		Write:  true,
		Delete: true,
	}
}
