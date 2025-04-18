package name_test

import (
	"github.com/cirglo.com/dfs/pkg/name"
	"testing"
)

type mockHasPermissions struct {
	permissions name.Permissions
}

func (m mockHasPermissions) GetPermissions() name.Permissions {
	return m.permissions
}

func TestPrincipal_ComputePrivileges(t *testing.T) {
	tests := []struct {
		name               string
		principal          name.Principal
		hasPermissionsList []name.HasPermissions
		expected           name.Privileges
	}{
		{
			name: "No permissions",
			principal: name.NewPrincipal(name.User{
				Name:   "user1",
				Groups: []*name.Group{},
			}),
			hasPermissionsList: []name.HasPermissions{},
			expected: name.Privileges{
				Read:   false,
				Write:  false,
				Delete: false,
			},
		},
		{
			name: "Owner permissions",
			principal: name.NewPrincipal(name.User{
				Name:   "user1",
				Groups: []*name.Group{},
			}),
			hasPermissionsList: []name.HasPermissions{
				mockHasPermissions{
					permissions: name.Permissions{
						Owner: "user1",
						OwnerPermission: name.Permission{
							Read:   true,
							Write:  true,
							Delete: true,
						},
					},
				},
			},
			expected: name.Privileges{
				Read:   true,
				Write:  true,
				Delete: true,
			},
		},
		{
			name: "Group permissions",
			principal: name.NewPrincipal(name.User{
				Name: "user1",
				Groups: []*name.Group{
					{Name: "group1"},
				},
			}),
			hasPermissionsList: []name.HasPermissions{
				mockHasPermissions{
					permissions: name.Permissions{
						Group: "group1",
						GroupPermission: name.Permission{
							Read:   true,
							Write:  false,
							Delete: true,
						},
					},
				},
			},
			expected: name.Privileges{
				Read:   true,
				Write:  false,
				Delete: true,
			},
		},
		{
			name:      "Root principal always has full privileges",
			principal: name.NewRootPrincipal(),
			hasPermissionsList: []name.HasPermissions{
				mockHasPermissions{
					permissions: name.Permissions{
						OwnerPermission: name.Permission{
							Read:   false,
							Write:  false,
							Delete: false,
						},
					},
				},
			},
			expected: name.Privileges{
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
