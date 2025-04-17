package name

import (
	"fmt"
	"gorm.io/gorm"
	"strings"
)

type HasPermissions interface {
	GetPermissions() Permissions
}

type Permission struct {
	Read   bool `gorm:"not null"`
	Write  bool `gorm:"not null"`
	Delete bool `gorm:"not null"`
}

type Permissions struct {
	Owner           string     `gorm:"not null"`
	Group           string     `gorm:"not null"`
	OwnerPermission Permission `gorm:"embedded;embeddedPrefix:owner_"`
	GroupPermission Permission `gorm:"embedded;embeddedPrefix:group_"`
	OtherPermission Permission `gorm:"embedded;embeddedPrefix:other_"`
}

func (p *Permissions) BeforeSave(_ *gorm.DB) error {
	p.Owner = strings.TrimSpace(p.Owner)
	p.Group = strings.TrimSpace(p.Group)

	if len(p.Owner) == 0 {
		return fmt.Errorf("owner is empty")
	}

	if len(p.Group) == 0 {
		return fmt.Errorf("group is empty")
	}

	return nil
}
