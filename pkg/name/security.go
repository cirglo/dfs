package name

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"github.com/sirupsen/logrus"
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID             uint64
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Name           string   `gorm:"uniqueIndex"`
	HashedPassword string   `gorm:"not null"`
	Groups         []*Group `gorm:"many2many:user_groups;"`
}

type Group struct {
	ID        uint64
	CreatedAt time.Time
	UpdatedAt time.Time
	Name      string  `gorm:"uniqueIndex"`
	Users     []*User `gorm:"many2many:user_groups;"`
}

type Token struct {
	ID        uint64
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt time.Time
	Value     string `gorm:"not null"`
	User      User   `gorm:"not null"`
}

func (t *Token) IsExpired() bool {
	return t.ExpiresAt.Before(time.Now())
}

type SecurityService interface {
	CreateUser(user User) error
	DeleteUser(userName string) error
	GetUser(userName string) (User, error)
	GetAllUsers() ([]User, error)
	GetGroup(groupName string) (Group, error)
	GetAllGroups() ([]Group, error)
	CreateGroup(group Group) error
	DeleteGroup(groupName string) error
	AddUserToGroup(userName string, groupName string) error
	RemoveUserFromGroup(userName string, groupName string) error
	AuthenticateUser(userName string, password string) (string, error)
	ChangeUserPassword(userName string, newPassword string) error
	Logout(token string) error
	LookupUserByToken(token string) (User, error)
}

type SecurityServiceOpts struct {
	Logger           *logrus.Logger
	DB               *gorm.DB
	TokenExperiation time.Duration
}

func (o *SecurityServiceOpts) Validate() error {
	if o.Logger == nil {
		return fmt.Errorf("logger is required")
	}
	if o.DB == nil {
		return fmt.Errorf("db is required")
	}
	if o.TokenExperiation == 0 {
		return fmt.Errorf("token expiration is required")
	}

	return nil
}

type securityService struct {
	Opts SecurityServiceOpts
}

func NewSecurityService(opts SecurityServiceOpts) (SecurityService, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("options are invalid: %w", err)
	}
	s := &securityService{
		Opts: opts,
	}
	return s, nil
}

func (s *securityService) CreateUser(user User) error {
	err := s.Opts.DB.Transaction(func(tx *gorm.DB) error {
		// Check if user already exists
		existingUser := User{}
		tx.Where("name = ?", user.Name).First(&existingUser)
		if tx.Error != nil && tx.RowsAffected > 0 {
			return fmt.Errorf("user %s already exists", user.Name)
		}

		// Create user
		tx.Create(&user)
		if tx.Error != nil {
			return fmt.Errorf("failed to create user: %w", tx.Error)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

func (s *securityService) DeleteUser(userName string) error {
	err := s.Opts.DB.Transaction(func(tx *gorm.DB) error {
		// Check if user exists
		user := User{}
		tx.Where("name = ?", userName).First(&user)
		if tx.Error != nil && tx.RowsAffected == 0 {
			return fmt.Errorf("user %s not found", userName)
		}

		// Delete user
		tx.Delete(&user)
		if tx.Error != nil {
			return fmt.Errorf("failed to delete user: %w", tx.Error)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

func (s *securityService) GetUser(userName string) (User, error) {
	user := User{}
	err := s.Opts.DB.Transaction(func(tx *gorm.DB) error {
		// Check if user exists
		tx.Where("name = ?", userName).First(&user)
		if tx.Error != nil && tx.RowsAffected == 0 {
			return fmt.Errorf("user %s not found", userName)
		}

		return nil
	}, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return User{}, fmt.Errorf("transaction failed: %w", err)
	}

	return user, nil
}

func (s *securityService) GetAllUsers() ([]User, error) {
	users := []User{}
	err := s.Opts.DB.Transaction(func(tx *gorm.DB) error {
		// Get all users
		tx.Find(&users)
		if tx.Error != nil {
			return fmt.Errorf("failed to get users: %w", tx.Error)
		}

		return nil
	}, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("transaction failed: %w", err)
	}

	return users, nil
}

func (s *securityService) GetGroup(groupName string) (Group, error) {
	group := Group{}
	err := s.Opts.DB.Transaction(func(tx *gorm.DB) error {
		// Check if group exists
		tx.Where("name = ?", groupName).First(&group)
		if tx.Error != nil && tx.RowsAffected == 0 {
			return fmt.Errorf("group %s not found", groupName)
		}

		return nil
	}, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return Group{}, fmt.Errorf("transaction failed: %w", err)
	}

	return group, nil
}

func (s *securityService) GetAllGroups() ([]Group, error) {
	groups := []Group{}
	err := s.Opts.DB.Transaction(func(tx *gorm.DB) error {
		// Get all groups
		tx.Find(&groups)
		if tx.Error != nil {
			return fmt.Errorf("failed to get groups: %w", tx.Error)
		}

		return nil
	}, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("transaction failed: %w", err)
	}

	return groups, nil
}

func (s *securityService) CreateGroup(group Group) error {
	err := s.Opts.DB.Transaction(func(tx *gorm.DB) error {
		// Check if group already exists
		existingGroup := Group{}
		tx.Where("name = ?", group.Name).First(&existingGroup)
		if tx.Error != nil && tx.RowsAffected > 0 {
			return fmt.Errorf("group %s already exists", group.Name)
		}

		// Create group
		tx.Create(&group)
		if tx.Error != nil {
			return fmt.Errorf("failed to create group: %w", tx.Error)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

func (s *securityService) DeleteGroup(groupName string) error {
	err := s.Opts.DB.Transaction(func(tx *gorm.DB) error {
		// Check if group exists
		group := Group{}
		tx.Where("name = ?", groupName).First(&group)
		if tx.Error != nil && tx.RowsAffected == 0 {
			return fmt.Errorf("group %s not found", groupName)
		}

		// Delete group
		tx.Delete(&group)
		if tx.Error != nil {
			return fmt.Errorf("failed to delete group: %w", tx.Error)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

func (s *securityService) AddUserToGroup(userName string, groupName string) error {
	err := s.Opts.DB.Transaction(func(tx *gorm.DB) error {
		// Check if group exists
		group := Group{}
		tx.Where("name = ?", groupName).First(&group)
		if tx.Error != nil && tx.RowsAffected == 0 {
			return fmt.Errorf("group %s not found", groupName)
		}

		// Check if user exists
		user := User{}
		tx.Where("name = ?", userName).First(&user)
		if tx.Error != nil && tx.RowsAffected == 0 {
			return fmt.Errorf("user %s not found", userName)
		}

		tx.Model(&group).Association("Users").Append(&user)
		if tx.Error != nil {
			return fmt.Errorf("failed to add user to group: %w", tx.Error)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

func (s *securityService) RemoveUserFromGroup(userName string, groupName string) error {
	err := s.Opts.DB.Transaction(func(tx *gorm.DB) error {
		// Check if group exists
		group := Group{}
		tx.Where("name = ?", groupName).First(&group)
		if tx.Error != nil && tx.RowsAffected == 0 {
			return fmt.Errorf("group %s not found", groupName)
		}

		// Check if user exists
		user := User{}
		tx.Where("name = ?", userName).First(&user)
		if tx.Error != nil && tx.RowsAffected == 0 {
			return fmt.Errorf("user %s not found", userName)
		}

		tx.Model(&group).Association("Users").Delete(&user)
		if tx.Error != nil {
			return fmt.Errorf("failed to remove user from group: %w", tx.Error)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

func (s *securityService) AuthenticateUser(userName string, password string) (string, error) {
	var t string
	err := s.Opts.DB.Transaction(func(tx *gorm.DB) error {
		user := User{}
		tx.Where("name = ?", userName).First(&user)
		if tx.Error != nil && tx.RowsAffected == 0 {
			return fmt.Errorf("user %s not found", userName)
		}

		if user.HashedPassword != password {
			return fmt.Errorf("invalid password for user %s", userName)
		}

		// Generate a token
		tokenBytes := make([]byte, 1024)
		_, err := rand.Read(tokenBytes)
		if err != nil {
			return fmt.Errorf("could not create token: %w", err)
		}
		tokenString := base64.StdEncoding.EncodeToString(tokenBytes)
		token := Token{
			Value:     tokenString,
			User:      user,
			ExpiresAt: time.Now().Add(s.Opts.TokenExperiation),
		}

		tx.Create(&token)
		if tx.Error != nil {
			return fmt.Errorf("failed to create token: %w", tx.Error)
		}

		t = tokenString

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("transaction failed: %w", err)
	}

	return t, nil
}

func (s *securityService) ChangeUserPassword(userName string, newPassword string) error {
	err := s.Opts.DB.Transaction(func(tx *gorm.DB) error {
		// Check if user exists
		user := User{}
		tx.Where("name = ?", userName).First(&user)
		if tx.Error != nil && tx.RowsAffected == 0 {
			return fmt.Errorf("user %s not found", userName)
		}

		// Update password
		user.HashedPassword = newPassword
		tx.Save(&user)
		if tx.Error != nil {
			return fmt.Errorf("failed to update user password: %w", tx.Error)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

func (s *securityService) Logout(token string) error {
	err := s.Opts.DB.Transaction(func(tx *gorm.DB) error {
		// Check if token exists
		tokenEntity := Token{}
		tx.Where("value = ?", token).First(&tokenEntity)
		if tx.Error != nil && tx.RowsAffected == 0 {
			return fmt.Errorf("token %s not found", token)
		}

		// Delete token
		tx.Delete(&tokenEntity)
		if tx.Error != nil {
			return fmt.Errorf("failed to delete token: %w", tx.Error)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

func (s *securityService) LookupUserByToken(token string) (User, error) {
	user := User{}
	err := s.Opts.DB.Transaction(func(tx *gorm.DB) error {
		// Check if token exists
		tokenEntity := Token{}
		tx.Where("value = ?", token).First(&tokenEntity)
		if tx.Error != nil {
			return fmt.Errorf("could not get token: %w", tx.Error)
		}

		if tokenEntity.IsExpired() {
			return fmt.Errorf("token is expired")
		}

		user = tokenEntity.User

		return nil
	}, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return User{}, fmt.Errorf("transaction failed: %w", err)
	}

	return user, nil
}

func (s *securityService) IsTokenValid(token string, userName string) bool {
	var valid bool = false

	err := s.Opts.DB.Transaction(func(tx *gorm.DB) error {
		// Check if token exists
		tokenEntity := Token{}
		tx.Where("value = ? AND user_id = (SELECT id FROM users WHERE name = ?)", token, userName).First(&tokenEntity)
		if tx.Error != nil {
			return fmt.Errorf("failed to check token: %w", tx.Error)
		}

		if tx.RowsAffected == 0 {
			valid = false
			return nil
		}

		if !tokenEntity.IsExpired() {
			valid = true
		}

		return nil
	}, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		s.Opts.Logger.Errorf("failed to check token %w", err)
		return false
	}

	return valid
}
