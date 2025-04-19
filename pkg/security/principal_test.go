package security_test

import (
	"github.com/cirglo.com/dfs/pkg/security"
	"testing"
)

type mockHasPermissions struct {
	permissions security.Permissions
}

func (m mockHasPermissions) GetPermissions() security.Permissions {
	return m.permissions
}

func TestPrincipal_ComputePrivileges(t *testing.T) {
	tests := []struct {
		name               string
		principal          security.Principal
		hasPermissionsList []security.HasPermissions
		expected           security.Privileges
	}{
		{
			name: "No permissions",
			principal: security.NewPrincipal(security.User{
				Name:   "user1",
				Groups: []*security.Group{},
			}),
			hasPermissionsList: []security.HasPermissions{},
			expected: security.Privileges{
				Read:   false,
				Write:  false,
				Delete: false,
			},
		},
		{
			name: "Owner permissions",
			principal: security.NewPrincipal(security.User{
				Name:   "user1",
				Groups: []*security.Group{},
			}),
			hasPermissionsList: []security.HasPermissions{
				mockHasPermissions{
					permissions: security.Permissions{
						Owner: "user1",
						OwnerPermission: security.Permission{
							Read:   true,
							Write:  true,
							Delete: true,
						},
					},
				},
			},
			expected: security.Privileges{
				Read:   true,
				Write:  true,
				Delete: true,
			},
		},
		{
			name: "Group permissions",
			principal: security.NewPrincipal(security.User{
				Name: "user1",
				Groups: []*security.Group{
					{Name: "group1"},
				},
			}),
			hasPermissionsList: []security.HasPermissions{
				mockHasPermissions{
					permissions: security.Permissions{
						Group: "group1",
						GroupPermission: security.Permission{
							Read:   true,
							Write:  false,
							Delete: true,
						},
					},
				},
			},
			expected: security.Privileges{
				Read:   true,
				Write:  false,
				Delete: true,
			},
		},
		{
			name:      "Root principal always has full privileges",
			principal: security.NewRootPrincipal(),
			hasPermissionsList: []security.HasPermissions{
				mockHasPermissions{
					permissions: security.Permissions{
						OwnerPermission: security.Permission{
							Read:   false,
							Write:  false,
							Delete: false,
						},
					},
				},
			},
			expected: security.Privileges{
				Read:   true,
				Write:  true,
				Delete: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.principal.ComputePrivileges(tt.hasPermissionsList...)
			if result != tt.expected {
				t.Errorf("ComputePrivileges() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
