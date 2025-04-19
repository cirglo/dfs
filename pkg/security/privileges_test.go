package security_test

import (
	"github.com/cirglo.com/dfs/pkg/security"
	"testing"
)

func TestPrivileges_Union(t *testing.T) {
	tests := []struct {
		name     string
		p1       security.Privileges
		p2       security.Privileges
		expected security.Privileges
	}{
		{
			name: "All true privileges",
			p1:   security.Privileges{Read: true, Write: true, Delete: true},
			p2:   security.Privileges{Read: true, Write: true, Delete: true},
			expected: security.Privileges{
				Read:   true,
				Write:  true,
				Delete: true,
			},
		},
		{
			name: "All false privileges",
			p1:   security.Privileges{Read: false, Write: false, Delete: false},
			p2:   security.Privileges{Read: false, Write: false, Delete: false},
			expected: security.Privileges{
				Read:   false,
				Write:  false,
				Delete: false,
			},
		},
		{
			name: "Mixed privileges - p1 true, p2 false",
			p1:   security.Privileges{Read: true, Write: true, Delete: true},
			p2:   security.Privileges{Read: false, Write: false, Delete: false},
			expected: security.Privileges{
				Read:   false,
				Write:  false,
				Delete: false,
			},
		},
		{
			name: "Mixed privileges - p1 false, p2 true",
			p1:   security.Privileges{Read: false, Write: false, Delete: false},
			p2:   security.Privileges{Read: true, Write: true, Delete: true},
			expected: security.Privileges{
				Read:   false,
				Write:  false,
				Delete: false,
			},
		},
		{
			name: "Partial privileges - Read true, others false",
			p1:   security.Privileges{Read: true, Write: false, Delete: false},
			p2:   security.Privileges{Read: true, Write: false, Delete: false},
			expected: security.Privileges{
				Read:   true,
				Write:  false,
				Delete: false,
			},
		},
		{
			name: "Partial privileges - Write true, others false",
			p1:   security.Privileges{Read: false, Write: true, Delete: false},
			p2:   security.Privileges{Read: false, Write: true, Delete: false},
			expected: security.Privileges{
				Read:   false,
				Write:  true,
				Delete: false,
			},
		},
		{
			name: "Partial privileges - Delete true, others false",
			p1:   security.Privileges{Read: false, Write: false, Delete: true},
			p2:   security.Privileges{Read: false, Write: false, Delete: true},
			expected: security.Privileges{
				Read:   false,
				Write:  false,
				Delete: true,
			},
		},
		{
			name: "Mixed privileges - different combinations",
			p1:   security.Privileges{Read: true, Write: false, Delete: true},
			p2:   security.Privileges{Read: false, Write: true, Delete: true},
			expected: security.Privileges{
				Read:   false,
				Write:  false,
				Delete: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.p1.Union(tt.p2)
			if result != tt.expected {
				t.Errorf("Union() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
