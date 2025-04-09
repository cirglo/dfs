package name

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

const (
	metaFileName = "meta.json"
)

type FileService interface {
	StatFile(p Principal, filePath string) (FileInfo, error)
	GetChildren(p Principal, filePath string) ([]FileInfo, error)
	GetParent(p Principal, filePath string) (FileInfo, error)
	CreateFile(p Principal, filePath string, perms Permissions) (FileInfo, error)
	CreateDir(p Principal, filePath string, perms Permissions) (FileInfo, error)
	DeleteFile(p Principal, filePath string) error
	DeleteDir(p Principal, filePath string) error
	AddBlockInfo(p Principal, filePath string, blockInfo BlockInfo) error
	RemoveBlockInfo(p Principal, filePath string, blockInfo BlockInfo) error
}

type FileInfo struct {
	Path        string      `json:"path"`
	IsDir       bool        `json:"isDir"`
	ID          string      `json:"id"`
	Size        uint64      `json:"size"`
	Owner       string      `json:"owner"`
	Group       string      `json:"group"`
	Permissions Permissions `json:"permissions"`
	BlockInfos  []BlockInfo `json:"blockInfos"`
}

type BlockInfo struct {
	ID        string     `json:"id"`
	Locations []Location `json:"locations"`
	Sequence  uint64     `json:"sequence"`
	Length    uint32     `json:"length"`
}

type Location struct {
	Hostname string `json:"hostname"`
	Port     uint16 `json:"port"`
}

type Principal struct {
	User  string
	Group string
}

func rootPrincipal() Principal {
	return Principal{
		User:  "",
		Group: "",
	}
}

func (p Principal) IsRoot() bool {
	return p.User == "" && p.Group == ""
}

type Privileges struct {
	Read  bool
	Write bool
}

func (p Privileges) Union(o Privileges) Privileges {
	return Privileges{
		Read:  p.Read && o.Read,
		Write: p.Write && o.Write,
	}
}

type Permission struct {
	Read  bool `json:"read"`
	Write bool `json:"write"`
}

type Permissions struct {
	Owner           string     `json:"owner"`
	Group           string     `json:"group"`
	OwnerPermission Permission `json:"ownerPermission"`
	GroupPermission Permission `json:"groupPermission"`
	OtherPermisson  Permission `json:"otherPermission"`
}

func (p Permissions) Privileges(principal Principal) Privileges {
	return Privileges{
		Read:  p.CanRead(principal),
		Write: p.CanWrite(principal),
	}
}

func (p Permissions) CanRead(principal Principal) bool {

	if principal.IsRoot() {
		return true
	}

	if p.OtherPermisson.Read {
		return true
	}

	if p.Group == principal.Group {
		if p.GroupPermission.Read {
			return true
		}
	}

	if p.Owner == principal.User {
		if p.OwnerPermission.Read {
			return true
		}
	}

	return false
}

func (p Permissions) CanWrite(principal Principal) bool {
	if principal.IsRoot() {
		return true
	}

	if p.OtherPermisson.Write {
		return true
	}

	if p.Group == principal.Group {
		if p.GroupPermission.Write {
			return true
		}
	}

	if p.Owner == principal.User {
		if p.OwnerPermission.Write {
			return true
		}
	}

	return false
}

type FileServiceOpts struct {
	Logger *logrus.Logger
	Dir    fs.FileInfo
}

func (f FileServiceOpts) Validate() error {
	if f.Logger == nil {
		return fmt.Errorf("logger is required")
	}
	if f.Dir == nil {
		return fmt.Errorf("dir is required")
	}
	if !f.Dir.IsDir() {
		return fmt.Errorf("dir must be a directory")
	}
	return nil
}

type fileService struct {
	Opts FileServiceOpts
	Lock sync.RWMutex
}

var _ FileService = &fileService{}

func NewFileService(opts FileServiceOpts) (FileService, error) {
	if err := opts.Validate(); err != nil {
		return nil, fmt.Errorf("invalid file service options: %w", err)
	}
	return &fileService{
		Opts: opts,
	}, nil
}

func (f *fileService) determinePrivileges(p Principal, filePath string) (Privileges, error) {
	if p.IsRoot() {
		return Privileges{
			Read:  true,
			Write: true,
		}, nil
	}

	realPath := filepath.Join(f.Opts.Dir.Name(), filePath)
	metaPath := filepath.Join(realPath, metaFileName)

	fi, err := f.readMetadataFile(metaPath)
	if err != nil {
		return Privileges{}, fmt.Errorf("failed to read metadata file %s: %w", metaPath, err)
	}

	privs := fi.Permissions.Privileges(p)

	if filePath != "/" && (privs.Read || privs.Write) {
		parentPath := filepath.Dir(realPath)
		parentPrivs, err := f.determinePrivileges(rootPrincipal(), parentPath)
		if err != nil {
			return Privileges{}, fmt.Errorf("failed to get parent priveleges %s: %w", parentPath, err)
		}

		privs = privs.Union(parentPrivs)
	}

	return privs, nil
}

func (f *fileService) readMetadataFile(path string) (FileInfo, error) {
	fi := FileInfo{}

	b, err := os.ReadFile(path)
	if err != nil {
		return fi, fmt.Errorf("failed to read metadata file %s: %w", path, err)
	}

	err = json.Unmarshal(b, &fi)
	if err != nil {
		return fi, fmt.Errorf("failed to unmarshal metadata file %s: %w", path, err)
	}

	return fi, nil
}

func (f *fileService) writeMetadataFile(path string, fi FileInfo) error {
	b, err := json.Marshal(fi)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata file %s: %w", path, err)
	}

	err = os.WriteFile(path, b, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to write metadata file %s: %w", path, err)
	}

	return nil
}

func (f *fileService) StatFile(p Principal, filePath string) (FileInfo, error) {
	f.Lock.RLock()
	defer f.Lock.RUnlock()
	privs, err := f.determinePrivileges(p, filePath)
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to determine privileges for %s: %w", filePath, err)
	}

	if !privs.Read {
		return FileInfo{}, fmt.Errorf("permission denied for %s", filePath)
	}

	realPath := filepath.Join(f.Opts.Dir.Name(), filePath)
	metaPath := filepath.Join(realPath, metaFileName)

	fi, err := f.readMetadataFile(metaPath)
	if err != nil {
		return fi, fmt.Errorf("failed to read metadata file %s: %w", metaPath, err)
	}

	return fi, nil
}

func (f *fileService) GetChildren(p Principal, filePath string) ([]FileInfo, error) {
	f.Lock.RLock()
	defer f.Lock.RUnlock()
	privs, err := f.determinePrivileges(p, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to determine privileges for %s: %w", filePath, err)
	}
	if !privs.Read {
		return nil, fmt.Errorf("permission denied for %s", filePath)
	}
	realPath := filepath.Join(f.Opts.Dir.Name(), filePath)
	dirs, err := os.ReadDir(realPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read dir %s: %w", realPath, err)
	}
	var children []FileInfo

	for _, dir := range dirs {
		if dir.IsDir() {
			metaPath := filepath.Join(realPath, dir.Name(), metaFileName)
			fi, err := f.readMetadataFile(metaPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read metadata file %s: %w", metaPath, err)
			}
			children = append(children, fi)
		}
	}

	return children, nil
}

func (f *fileService) GetParent(p Principal, path string) (FileInfo, error) {
	f.Lock.RLock()
	defer f.Lock.RUnlock()
	privs, err := f.determinePrivileges(p, filepath.Dir(path))
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to determine privileges for %s: %w", path, err)
	}
	if !privs.Read {
		return FileInfo{}, fmt.Errorf("permission denied for %s", path)
	}
	realPath := filepath.Join(f.Opts.Dir.Name(), path)
	parentPath := filepath.Dir(realPath)
	metaPath := filepath.Join(parentPath, metaFileName)

	fi, err := f.readMetadataFile(metaPath)
	if err != nil {
		return fi, fmt.Errorf("failed to read metadata file %s: %w", metaPath, err)
	}

	return fi, nil
}

func (f *fileService) CreateFile(p Principal, path string, perms Permissions) (FileInfo, error) {
	f.Lock.Lock()
	defer f.Lock.Unlock()
	privs, err := f.determinePrivileges(p, filepath.Dir(path))
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to determine privileges for %s: %w", path, err)
	}
	if !privs.Write {
		return FileInfo{}, fmt.Errorf("permission denied for %s", path)
	}
	realPath := filepath.Join(f.Opts.Dir.Name(), path)

	err = os.MkdirAll(realPath, os.ModePerm)
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to create dir %s: %w", realPath, err)
	}

	metaPath := filepath.Join(realPath, metaFileName)
	fi := FileInfo{
		Path:        path,
		IsDir:       false,
		ID:          uuid.New().String(),
		Size:        0,
		Permissions: perms,
		BlockInfos:  []BlockInfo{},
	}

	err = f.writeMetadataFile(metaPath, fi)
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to write metadata file %s: %w", metaPath, err)
	}

	return fi, nil
}

func (f *fileService) CreateDir(p Principal, path string, perms Permissions) (FileInfo, error) {
	f.Lock.Lock()
	defer f.Lock.Unlock()
	privs, err := f.determinePrivileges(p, filepath.Dir(path))
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to determine privileges for %s: %w", path, err)
	}
	if !privs.Write {
		return FileInfo{}, fmt.Errorf("permission denied for %s", path)
	}
	realPath := filepath.Join(f.Opts.Dir.Name(), path)

	err = os.MkdirAll(realPath, os.ModePerm)
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to create dir %s: %w", realPath, err)
	}

	metaPath := filepath.Join(realPath, metaFileName)
	fi := FileInfo{
		Path:        path,
		IsDir:       true,
		ID:          uuid.New().String(),
		Size:        0,
		Permissions: perms,
		BlockInfos:  []BlockInfo{},
	}

	err = f.writeMetadataFile(metaPath, fi)
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to write metadata file %s: %w", metaPath, err)
	}

	return fi, nil
}

func (f *fileService) DeleteFile(p Principal, filePath string) error {
	f.Lock.Lock()
	defer f.Lock.Unlock()
	privs, err := f.determinePrivileges(p, filePath)
	if err != nil {
		return fmt.Errorf("failed to determine privileges for %s: %w", filePath, err)
	}
	if !privs.Write {
		return fmt.Errorf("permission denied for %s", filePath)
	}
	realPath := filepath.Join(f.Opts.Dir.Name(), filePath)
	metaPath := filepath.Join(realPath, metaFileName)
	fi, err := f.readMetadataFile(metaPath)
	if err != nil {
		return fmt.Errorf("failed to read metadata file %s: %w", metaPath, err)
	}

	if fi.IsDir {
		return fmt.Errorf("cannot delete directory %s", filePath)
	}

	err = os.Remove(metaPath)
	if err != nil {
		return fmt.Errorf("failed to remove metadata file %s: %w", metaPath, err)
	}

	err = os.Remove(realPath)
	if err != nil {
		return fmt.Errorf("failed to remove file %s: %w", realPath, err)
	}

	return nil
}

func (f *fileService) DeleteDir(p Principal, filePath string) error {
	f.Lock.Lock()
	defer f.Lock.Unlock()
	privs, err := f.determinePrivileges(p, filePath)
	if err != nil {
		return fmt.Errorf("failed to determine privileges for %s: %w", filePath, err)
	}
	if !privs.Write {
		return fmt.Errorf("permission denied for %s", filePath)
	}
	realPath := filepath.Join(f.Opts.Dir.Name(), filePath)
	metaPath := filepath.Join(realPath, metaFileName)
	fi, err := f.readMetadataFile(metaPath)
	if err != nil {
		return fmt.Errorf("failed to read metadata file %s: %w", metaPath, err)
	}

	if !fi.IsDir {
		return fmt.Errorf("cannot delete file %s", filePath)
	}

	dirs, err := os.ReadDir(realPath)
	if err != nil {
		return fmt.Errorf("failed to read dir %s: %w", realPath, err)
	}

	if len(dirs) > 0 {
		return fmt.Errorf("directory %s is not empty", filePath)
	}

	err = os.Remove(metaPath)
	if err != nil {
		return fmt.Errorf("failed to remove metadata file %s: %w", metaPath, err)
	}

	err = os.Remove(realPath)
	if err != nil {
		return fmt.Errorf("failed to remove file %s: %w", realPath, err)
	}

	return nil
}

func (f *fileService) AddBlockInfo(p Principal, filePath string, blockInfo BlockInfo) error {
	f.Lock.Lock()
	defer f.Lock.Unlock()
	privs, err := f.determinePrivileges(p, filePath)
	if err != nil {
		return fmt.Errorf("failed to determine privileges for %s: %w", filePath, err)
	}
	if !privs.Write {
		return fmt.Errorf("permission denied for %s", filePath)
	}
	realPath := filepath.Join(f.Opts.Dir.Name(), filePath)
	metaPath := filepath.Join(realPath, metaFileName)
	fi, err := f.readMetadataFile(metaPath)
	if err != nil {
		return fmt.Errorf("failed to read metadata file %s: %w", metaPath, err)
	}

	fi.BlockInfos = append(fi.BlockInfos, blockInfo)

	sort.Slice(fi.BlockInfos, func(i, j int) bool {
		return fi.BlockInfos[i].ID < fi.BlockInfos[j].ID
	})

	return nil
}

func (f *fileService) RemoveBlockInfo(p Principal, filePath string, blockInfo BlockInfo) error {
	f.Lock.Lock()
	defer f.Lock.Unlock()
	privs, err := f.determinePrivileges(p, filePath)
	if err != nil {
		return fmt.Errorf("failed to determine privileges for %s: %w", filePath, err)
	}
	if !privs.Write {
		return fmt.Errorf("permission denied for %s", filePath)
	}
	realPath := filepath.Join(f.Opts.Dir.Name(), filePath)
	metaPath := filepath.Join(realPath, metaFileName)
	fi, err := f.readMetadataFile(metaPath)
	if err != nil {
		return fmt.Errorf("failed to read metadata file %s: %w", metaPath, err)
	}

	var blockInfos []BlockInfo

	for _, bi := range fi.BlockInfos {
		if bi.ID != blockInfo.ID {
			blockInfos = append(blockInfos, bi)
		}
	}

	fi.BlockInfos = blockInfos

	return nil
}
