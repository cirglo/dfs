package name

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"slices"
	"sync"
	"sync/atomic"
)

type Data struct {
	Users  []User  `json:"users"`
	Groups []Group `json:"groups"`
}

type User struct {
	Name           string `json:"name"`
	HashedPassword string `json:"hashed_password"`
	HashType       string `json:"hash_type"`
}

type Group struct {
	Name  string   `json:"name"`
	Users []string `json:"users"`
}

type SecurityService interface {
	CreateUser(user User) error
	DeleteUser(userName string) error
	GetUser(userName string) (User, error)
	GetAllUsers() ([]User, error)
	GetUserGroups(userName string) ([]Group, error)
	GetGroup(groupName string) (Group, error)
	GetAllGroups() ([]Group, error)
	CreateGroup(group Group) error
	DeleteGroup(groupName string) error
	AddUserToGroup(userName string, groupName string) error
	RemoveUserFromGroup(userName string, groupName string) error
	AuthenticateUser(userName string, hashType string, password string) (string, error)
	ChangeUserPassword(userName string, hashType string, newPassword string) error
	Logout(token string) error
	IsTokenValid(token string, userName string) bool
}

type SecurityServiceOpts struct {
	Logger   *logrus.Logger
	FilePath string
}

func (o *SecurityServiceOpts) Validate() error {
	if o.Logger == nil {
		return fmt.Errorf("logger is required")
	}
	if o.FilePath == "" {
		return fmt.Errorf("file path is required")
	}
	return nil
}

type securityService struct {
	Opts   SecurityServiceOpts
	Data   atomic.Value
	Lock   sync.RWMutex
	Tokens map[string]string
}

func NewSecurityService(opts SecurityServiceOpts) (SecurityService, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("options are invalid: %w", err)
	}
	s := &securityService{
		Opts:   opts,
		Data:   atomic.Value{},
		Lock:   sync.RWMutex{},
		Tokens: map[string]string{},
	}
	s.read()
	return s, nil
}

func (s *securityService) read() error {
	b, err := os.ReadFile(s.Opts.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			s.Opts.Logger.Info("file does not exist, creating new data")
			s.Data.Store(Data{})
			return nil
		}
		return fmt.Errorf("failed to read file: %w", err)
	}

	data := Data{}
	err = json.Unmarshal(b, &data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	s.Opts.Logger.Info("data loaded successfully")
	s.Data.Store(data)
	return nil
}

func (s *securityService) write() error {
	data := s.Data.Load().(Data)
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	err = os.WriteFile(s.Opts.FilePath, b, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	s.Opts.Logger.Info("data written successfully")
	return nil
}

func (s *securityService) currentData() Data {
	return s.Data.Load().(Data)
}

func (s *securityService) CreateUser(user User) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	data := s.currentData()

	for _, u := range data.Users {
		if u.Name == user.Name {
			return fmt.Errorf("user %s already exists", user.Name)
		}
	}
	data.Users = append(data.Users, user)
	err := s.write()
	if err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}
	return nil
}

func (s *securityService) DeleteUser(userName string) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	data := s.currentData()

	slices.DeleteFunc(data.Users, func(u User) bool {
		return u.Name == userName
	})

	err := s.write()
	if err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	return nil
}

func (s *securityService) GetUser(userName string) (User, error) {
	s.Lock.RLock()
	defer s.Lock.RUnlock()
	data := s.currentData()

	for _, u := range data.Users {
		if u.Name == userName {
			return u, nil
		}
	}
	return User{}, fmt.Errorf("user %s not found", userName)
}

func (s *securityService) GetAllUsers() ([]User, error) {
	s.Lock.RLock()
	defer s.Lock.RUnlock()
	u := s.currentData().Users
	c := make([]User, len(u))
	copy(c, u)

	return c, nil
}

func (s *securityService) GetUserGroups(userName string) ([]Group, error) {
	s.Lock.RLock()
	defer s.Lock.RUnlock()
	data := s.currentData()

	var groups []Group
	for _, g := range data.Groups {
		if slices.Contains(g.Users, userName) {
			groups = append(groups, g)
		}
	}
	return groups, nil
}

func (s *securityService) GetGroup(groupName string) (Group, error) {
	s.Lock.RLock()
	defer s.Lock.RUnlock()
	data := s.currentData()

	for _, g := range data.Groups {
		if g.Name == groupName {
			return g, nil
		}
	}

	return Group{}, fmt.Errorf("group %s not found", groupName)
}

func (s *securityService) GetAllGroups() ([]Group, error) {
	s.Lock.RLock()
	defer s.Lock.RUnlock()
	g := s.currentData().Groups
	c := make([]Group, len(g))
	copy(c, g)

	return c, nil
}

func (s *securityService) CreateGroup(group Group) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	data := s.currentData()

	for _, g := range data.Groups {
		if g.Name == group.Name {
			return fmt.Errorf("group %s already exists", group.Name)
		}
	}
	data.Groups = append(data.Groups, group)
	err := s.write()
	if err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}
	return nil
}

func (s *securityService) DeleteGroup(groupName string) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	data := s.currentData()

	slices.DeleteFunc(data.Groups, func(g Group) bool {
		return g.Name == groupName
	})

	err := s.write()
	if err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	return nil
}

func (s *securityService) AddUserToGroup(userName string, groupName string) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	data := s.currentData()

	for i, g := range data.Groups {
		if g.Name == groupName {
			if slices.Contains(g.Users, userName) {
				return fmt.Errorf("user %s already in group %s", userName, groupName)
			}
			data.Groups[i].Users = append(data.Groups[i].Users, userName)
			break
		}
	}

	err := s.write()
	if err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	return nil
}

func (s *securityService) RemoveUserFromGroup(userName string, groupName string) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	data := s.currentData()

	for i, g := range data.Groups {
		if g.Name == groupName {
			if !slices.Contains(g.Users, userName) {
				return fmt.Errorf("user %s not in group %s", userName, groupName)
			}
			slices.DeleteFunc(data.Groups[i].Users, func(u string) bool {
				return u == userName
			})
			break
		}
	}

	err := s.write()
	if err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	return nil
}

func (s *securityService) AuthenticateUser(userName string, hashType string, password string) (string, error) {
	s.Lock.RLock()
	defer s.Lock.RUnlock()
	data := s.currentData()

	for _, u := range data.Users {
		if u.Name == userName {
			if u.HashedPassword != password {
				return "", fmt.Errorf("invalid password for user %s", userName)
			}

			token := make([]byte, 128)
			_, err := rand.Read(token)
			if err != nil {
				return "", fmt.Errorf("could not create token: %w", err)
			}

			tokenString := base64.StdEncoding.EncodeToString(token)

			s.Tokens[tokenString] = userName

			return tokenString, nil
		}
	}

	return "", fmt.Errorf("user %s not found", userName)
}

func (s *securityService) ChangeUserPassword(userName string, hashType string, newPassword string) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()
	data := s.currentData()

	for i, u := range data.Users {
		if u.Name == userName {
			data.Users[i].HashedPassword = newPassword
			break
		}
	}

	err := s.write()
	if err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	return nil
}

func (s *securityService) Logout(token string) error {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	if _, ok := s.Tokens[token]; !ok {
		return fmt.Errorf("invalid token")
	}

	delete(s.Tokens, token)
	return nil
}

func (s *securityService) IsTokenValid(token string, userName string) bool {
	s.Lock.RLock()
	defer s.Lock.RUnlock()

	if u, ok := s.Tokens[token]; ok {
		return u == userName
	}
	return false
}
