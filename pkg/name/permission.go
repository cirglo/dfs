package name

type HasPermissions interface {
	GetPermissions() Permissions
}

type Permission struct {
	Read   bool `gorm:"column:can_read;not null"`
	Write  bool `gorm:"column:can_write;not null"`
	Delete bool `gorm:"column:can_delete;notnull""`
}

type Permissions struct {
	Owner           string     `gorm:"column:owner;not null"`
	Group           string     `gorm:"column:group;not null"`
	OwnerPermission Permission `gorm:"embedded;embeddedPrefix:owner_"`
	GroupPermission Permission `gorm:"embedded;emebeddedPrefix:group_"`
	OtherPermission Permission `gorm:"embedded;embeddedPrefix:other_"`
}
