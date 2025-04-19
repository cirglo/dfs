package security_test

import (
	"github.com/cirglo.com/dfs/pkg/security"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestPermissions_BeforeSave_Valid(t *testing.T) {
	permissions := &security.Permissions{
		Owner: "  valid_owner  ",
		Group: "  valid_group  ",
	}

	err := permissions.BeforeSave(&gorm.DB{})
	assert.NoError(t, err)
	assert.Equal(t, "valid_owner", permissions.Owner)
	assert.Equal(t, "valid_group", permissions.Group)
}

func TestPermissions_BeforeSave_EmptyOwner(t *testing.T) {
	permissions := &security.Permissions{
		Owner: "   ",
		Group: "valid_group",
	}

	err := permissions.BeforeSave(&gorm.DB{})
	assert.Error(t, err)
	assert.EqualError(t, err, "owner is empty")
}

func TestPermissions_BeforeSave_EmptyGroup(t *testing.T) {
	permissions := &security.Permissions{
		Owner: "valid_owner",
		Group: "   ",
	}

	err := permissions.BeforeSave(&gorm.DB{})
	assert.Error(t, err)
	assert.EqualError(t, err, "group is empty")
}
