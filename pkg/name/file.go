package name

import (
	"database/sql"
	"fmt"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"
)

type FileService interface {
	StatFile(p Principal, filePath string) (FileInfo, error)
	ListFiles(p Principal, path string) ([]FileInfo, error)
	ListDirs(p Principal, path string) ([]DirInfo, error)
	CreateFile(p Principal, path string, perms Permissions) (FileInfo, error)
	CreateDir(p Principal, path string, perms Permissions) (FileInfo, error)
	DeleteFile(p Principal, path string) error
	DeleteDir(p Principal, path string) error
	UpsertBlockInfos(p Principal, path string, blockInfos []BlockInfo) error
	GetBlockInfos(p Principal, path string) ([]BlockInfo, error)
}

type DirInfo struct {
	ID          uint64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Name        string      `gorm:"uniqueIndex:idx_dirinfo_name;not null;not empty"`
	Parent      *DirInfo    `gorm:"many2many:dir_childdirs;uniqueIndex:idx_dirinfo_name;"`
	ChildDirs   []*DirInfo  `gorm:"many2many:dir_childdirs;not null"`
	ChildFiles  []*FileInfo `gorm:"many2many:dir_childfiles;not null"`
	Permissions Permissions `gorm:"embedded;not null;"`
}

type FileInfo struct {
	ID          uint64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Name        string      `gorm:"uniqueIndex:idx_fileinfo_name;not null;not empty;"`
	Dir         *DirInfo    `gorm:"many2many:dir_childfiles;uniqueIndex:idx_fileinfo_name;not null;"`
	Size        uint64      `gorm:"not null;"`
	Permissions Permissions `gorm:"embedded;not null;"`
	BlockInfos  []BlockInfo
	Healthy     bool `gorm:"not null;"`
}

type BlockInfo struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	Locations []string
	Sequence  uint64 `gorm:"uniqueIndex:idx_fileinfo_sequence;not null"`
	Length    uint32 `gorm:"not null;"`
	CRC       uint32 `gorm:"not null;"`
}

type Principal interface {
	ComputePrivileges(permissions Permissions) Privileges
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

func (p principal) ComputePrivileges(permissions Permissions) Privileges {
	canRead := false
	canWrite := false

	if permissions.OtherPermisson.Read {
		canRead = true
	}

	if permissions.OtherPermisson.Write {
		canWrite = true
	}

	if canRead && canWrite {
		return Privileges{
			Read:  true,
			Write: true,
		}
	}

	if permissions.Owner == p.user {
		if permissions.OwnerPermission.Read {
			canRead = true
		}

		if permissions.OwnerPermission.Write {
			canWrite = true
		}
	}

	if canRead && canWrite {
		return Privileges{
			Read:  true,
			Write: true,
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

			if canRead && canWrite {
				return Privileges{
					Read:  true,
					Write: true,
				}
			}
		}
	}

	return Privileges{
		Read:  canRead,
		Write: canWrite,
	}

}

type rootPrincipal struct {
}

func NewRootPrincipal() Principal {
	return &rootPrincipal{}
}

func (p rootPrincipal) ComputePrivileges(permissions Permissions) Privileges {
	return Privileges{
		Read:  true,
		Write: true,
	}
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
	Read  bool `gorm:"not null;"`
	Write bool `gorm:"not null;"`
}

type Permissions struct {
	Owner           string     `gorm:"not null;not empty;"`
	Group           string     `gorm:"not null;not empty;"`
	OwnerPermission Permission `gorm:"not null;"`
	GroupPermission Permission `gorm:"not null;"`
	OtherPermisson  Permission `gorm:"not null;"`
}

type FileServiceOpts struct {
	Logger *logrus.Logger
	DB     *gorm.DB
}

func (f FileServiceOpts) Validate() error {
	if f.Logger == nil {
		return fmt.Errorf("logger is required")
	}
	if f.DB == nil {
		return fmt.Errorf("db is required")
	}
	return nil
}

type fileService struct {
	Opts FileServiceOpts
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

func (f *fileService) lookupDirs(tx *gorm.DB, p Principal, path string) ([]DirInfo, Privileges, error) {
	var dirInfos []DirInfo
	var parent *DirInfo
	privs := Privileges{
		Read:  false,
		Write: false,
	}

	for name := range strings.Split(path, "/") {
		current := DirInfo{}

		if parent == nil {
			err := tx.Where("parent IS NULL").First(&current).Error
			if err != nil {
				return dirInfos, privs, fmt.Errorf("failed to get root dir: %w", err)
			}
			privs = p.ComputePrivileges(current.Permissions)
		} else {
			err := tx.Where("parent = ? AND name = ?", parent.ID, name).First(&current).Error
			if err != nil {
				return dirInfos, privs, fmt.Errorf("failed to get child dir %s: %w", name, err)
			}
			privs = p.ComputePrivileges(current.Permissions).Union(privs)
		}

		dirInfos = append(dirInfos, current)
		parent = &current

	}

	return dirInfos, privs, nil
}

func (f *fileService) lookupFile(tx *gorm.DB, p Principal, path string) ([]DirInfo, FileInfo, Privileges, error) {
	dirInfos, privs, err := f.lookupDirs(tx, p, filepath.Dir(path))
	if err != nil {
		return nil, FileInfo{}, Privileges{}, fmt.Errorf("failed to lookup dirs: %w", err)
	}
	fileInfo := FileInfo{}
	err = tx.Where("parent = ? AND name = ?", dirInfos[len(dirInfos)-1].ID, filepath.Base(path)).First(&fileInfo).Error
	if err != nil {
		return nil, FileInfo{}, Privileges{}, fmt.Errorf("failed to get file %s: %w", path, err)
	}

	privs = privs.Union(p.ComputePrivileges(fileInfo.Permissions))

	return dirInfos, fileInfo, privs, nil
}

func (f *fileService) StatFile(p Principal, filePath string) (FileInfo, error) {
	var fileInfo FileInfo
	err := f.Opts.DB.Transaction(func(tx *gorm.DB) error {
		_, fi, privs, err := f.lookupFile(tx, p, filePath)
		if err != nil {
			return fmt.Errorf("failed to lookup file %s: %w", filePath, err)
		}

		if !privs.Read {
			return fmt.Errorf("permission denied for %s", filePath)
		}

		fileInfo = fi

		return nil
	}, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	return fileInfo, nil
}

func (f *fileService) StatDir(p Principal, dirPath string) (DirInfo, error) {
	var dirInfo DirInfo
	err := f.Opts.DB.Transaction(func(tx *gorm.DB) error {
		dirInfos, privs, err := f.lookupDirs(tx, p, dirPath)
		if err != nil {
			return fmt.Errorf("failed to lookup dir %s: %w", dirPath, err)
		}

		if !privs.Read {
			return fmt.Errorf("permission denied for %s", dirPath)
		}

		dirInfo = dirInfos[len(dirInfos)-1]

		return nil
	}, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return DirInfo{}, fmt.Errorf("failed to stat dir %s: %w", dirPath, err)
	}

	return dirInfo, nil
}

func (f *fileService) ListFiles(p Principal, path string) ([]FileInfo, error) {
	var fileInfos []FileInfo
	err := f.Opts.DB.Transaction(func(tx *gorm.DB) error {
		dirInfos, privs, err := f.lookupDirs(tx, p, path)
		if err != nil {
			return fmt.Errorf("failed to lookup dirs: %w", err)
		}

		if !privs.Read {
			return fmt.Errorf("permission denied for %s", path)
		}

		err = tx.Model(&dirInfos[len(dirInfos)-1]).Association("ChildFiles").Find(&fileInfos)
		if err != nil {
			return fmt.Errorf("failed to list files: %w", err)
		}

		return nil
	}, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].Name < fileInfos[j].Name
	})

	return fileInfos, nil
}

func (f *fileService) ListDirs(p Principal, path string) ([]DirInfo, error) {
	var dirInfos []DirInfo
	err := f.Opts.DB.Transaction(func(tx *gorm.DB) error {
		dirInfos, privs, err := f.lookupDirs(tx, p, path)
		if err != nil {
			return fmt.Errorf("failed to lookup dirs: %w", err)
		}

		if !privs.Read {
			return fmt.Errorf("permission denied for %s", path)
		}

		err = tx.Model(&dirInfos[len(dirInfos)-1]).Association("ChildDirs").Find(&dirInfos)
		if err != nil {
			return fmt.Errorf("failed to list dirs: %w", err)
		}

		return nil
	}, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list dirs: %w", err)
	}

	sort.Slice(dirInfos, func(i, j int) bool {
		return dirInfos[i].Name < dirInfos[j].Name
	})

	return dirInfos, nil
}

func (f *fileService) CreateFile(p Principal, path string, perms Permissions) (FileInfo, error) {
	var fileInfo FileInfo
	err := f.Opts.DB.Transaction(func(tx *gorm.DB) error {
		dirInfos, privs, err := f.lookupDirs(tx, p, path)
		if err != nil {
			return fmt.Errorf("failed to lookup dirs: %w", err)
		}

		if !privs.Write {
			return fmt.Errorf("permission denied for %s", path)
		}

		parentDir := dirInfos[len(dirInfos)-1]
		fileInfo = FileInfo{
			Name:        filepath.Base(path),
			Dir:         &parentDir,
			Size:        0,
			Permissions: perms,
			BlockInfos:  []BlockInfo{},
			Healthy:     true,
		}

		contains := slices.ContainsFunc(parentDir.ChildFiles, func(f *FileInfo) bool {
			return f.Name == fileInfo.Name
		})

		if contains {
			return fmt.Errorf("file %s already exists", path)
		}

		contains = slices.ContainsFunc(parentDir.ChildDirs, func(d *DirInfo) bool {
			return d.Name == fileInfo.Name
		})

		if contains {
			return fmt.Errorf("file %s is a directory", path)
		}

		err = tx.Create(&fileInfo).Error
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", path, err)
		}

		return nil
	})
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to create file %s: %w", path, err)
	}

	return fileInfo, nil
}

func (f *fileService) CreateDir(p Principal, path string, perms Permissions) (FileInfo, error) {
	var dirInfo DirInfo
	err := f.Opts.DB.Transaction(func(tx *gorm.DB) error {
		dirInfos, privs, err := f.lookupDirs(tx, p, path)
		if err != nil {
			return fmt.Errorf("failed to lookup dirs: %w", err)
		}

		if !privs.Write {
			return fmt.Errorf("permission denied for %s", path)
		}

		parentDir := dirInfos[len(dirInfos)-1]
		dirInfo = DirInfo{
			Name:        filepath.Base(path),
			Permissions: perms,
			Parent:      &parentDir,
		}

		contains := slices.ContainsFunc(parentDir.ChildDirs, func(d *DirInfo) bool {
			return d.Name == dirInfo.Name
		})

		if contains {
			return fmt.Errorf("directory %s already exists", path)
		}

		contains = slices.ContainsFunc(parentDir.ChildFiles, func(f *FileInfo) bool {
			return f.Name == dirInfo.Name
		})

		if contains {
			return fmt.Errorf("directory %s is a file", path)
		}

		err = tx.Create(&dirInfo).Error
		if err != nil {
			return fmt.Errorf("failed to create dir %s: %w", path, err)
		}

		return nil
	})
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to create dir %s: %w", path, err)
	}

	return FileInfo{}, nil
}

func (f *fileService) DeleteFile(p Principal, path string) error {
	err := f.Opts.DB.Transaction(func(tx *gorm.DB) error {
		dirInfos, fileInfo, privs, err := f.lookupFile(tx, p, path)
		if err != nil {
			return fmt.Errorf("failed to lookup file %s: %w", path, err)
		}

		if !privs.Write {
			return fmt.Errorf("permission denied for %s", path)
		}

		err = tx.Model(&dirInfos[len(dirInfos)-1]).Association("ChildFiles").Delete(&fileInfo)
		if err != nil {
			return fmt.Errorf("failed to delete file %s: %w", path, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to delete file %s: %w", path, err)
	}

	return nil
}

func (f *fileService) DeleteDir(p Principal, path string) error {
	err := f.Opts.DB.Transaction(func(tx *gorm.DB) error {
		dirInfos, privs, err := f.lookupDirs(tx, p, path)
		if err != nil {
			return fmt.Errorf("failed to lookup dirs: %w", err)
		}

		if !privs.Write {
			return fmt.Errorf("permission denied for %s", path)
		}

		dirInfo := dirInfos[len(dirInfos)-1]
		if len(dirInfo.ChildFiles) > 0 {
			return fmt.Errorf("directory %s is not empty", path)
		}
		if len(dirInfo.ChildDirs) > 0 {
			return fmt.Errorf("directory %s is not empty", path)
		}

		err = tx.Delete(&dirInfo).Error
		if err != nil {
			return fmt.Errorf("failed to delete dir %s: %w", path, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to delete dir %s: %w", path, err)
	}

	return nil
}

func (f *fileService) validateBlockInfos(tx *gorm.DB, fileInfo *FileInfo) error {
	blockInfos := append([]BlockInfo{}, fileInfo.BlockInfos...)
	sequenceMap := map[uint64]*BlockInfo{}

	sort.Slice(blockInfos, func(i, j int) bool {
		return blockInfos[i].Sequence < blockInfos[j].Sequence
	})

	for _, blockInfo := range blockInfos {
		if _, ok := sequenceMap[blockInfo.Sequence]; ok {
			return fmt.Errorf("duplicate block info sequence %d", blockInfo.Sequence)
		}
		sequenceMap[blockInfo.Sequence] = &blockInfo
	}

	totalLength := uint64(0)
	for i := 0; i < len(blockInfos)-1; i++ {
		if blockInfos[i].Sequence+1 != blockInfos[i+1].Sequence {
			fileInfo.Healthy = false
		}
		totalLength += uint64(blockInfos[i].Length)
	}

	fileInfo.Size = totalLength

	err := tx.Updates(fileInfo).Error
	if err != nil {
		return fmt.Errorf("failed to update file info: %w", err)
	}

	return nil
}

func (f *fileService) GetBlockInfos(p Principal, path string) ([]BlockInfo, error) {
	var blockInfos []BlockInfo
	err := f.Opts.DB.Transaction(func(tx *gorm.DB) error {
		_, fileInfo, privs, err := f.lookupFile(tx, p, path)
		if err != nil {
			return fmt.Errorf("failed to lookup file %s: %w", path, err)
		}

		if !privs.Read {
			return fmt.Errorf("permission denied for %s", path)
		}

		err = tx.Model(&fileInfo).Association("BlockInfos").Find(&blockInfos)
		if err != nil {
			return fmt.Errorf("failed to get block info: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get block info for file %s: %w", path, err)
	}

	sort.Slice(blockInfos, func(i, j int) bool {
		return blockInfos[i].Sequence < blockInfos[j].Sequence
	})

	return blockInfos, nil
}

func (f *fileService) UpsertBlockInfos(p Principal, path string, blockInfos []BlockInfo) error {
	incomingIndex := map[string]BlockInfo{}
	for _, blockInfo := range blockInfos {
		incomingIndex[blockInfo.ID] = blockInfo
	}

	err := f.Opts.DB.Transaction(func(tx *gorm.DB) error {
		_, fileInfo, privs, err := f.lookupFile(tx, p, path)
		if err != nil {
			return fmt.Errorf("failed to lookup file %s: %w", path, err)
		}

		if !privs.Write {
			return fmt.Errorf("permission denied for %s", path)
		}

		for _, blockInfo := range fileInfo.BlockInfos {
			incoming, ok := incomingIndex[blockInfo.ID]
			if ok {
				locationsToAdd := []string{}
				for _, location := range incoming.Locations {
					if !slices.Contains(blockInfo.Locations, location) {
						locationsToAdd = append(locationsToAdd, location)
					}
				}

				err := tx.Model(&blockInfo).Association("Locations").Append(locationsToAdd)
				if err != nil {
					return fmt.Errorf("failed to append block info locations: %w", err)
				}
			}

			delete(incomingIndex, blockInfo.ID)
		}

		for _, blockInfo := range incomingIndex {
			err := tx.Model(&fileInfo).Association("BlockInfos").Append(blockInfo)
			if err != nil {
				return fmt.Errorf("failed to append block info: %w", err)
			}
		}

		err = f.validateBlockInfos(tx, &fileInfo)
		if err != nil {
			return fmt.Errorf("failed to validate block info: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to upsert block info for file %s: %w", path, err)
	}

	return nil
}

func (f *fileService) RemoveBlockInfos(p Principal, path string, blockInfos []BlockInfo) error {
	incomingIndex := map[string]BlockInfo{}
	for _, blockInfo := range blockInfos {
		incomingIndex[blockInfo.ID] = blockInfo
	}

	err := f.Opts.DB.Transaction(func(tx *gorm.DB) error {
		_, fileInfo, privs, err := f.lookupFile(tx, p, path)
		if err != nil {
			return fmt.Errorf("failed to lookup file %s: %w", path, err)
		}

		if !privs.Write {
			return fmt.Errorf("permission denied for %s", path)
		}

		for _, blockInfo := range fileInfo.BlockInfos {
			incoming, ok := incomingIndex[blockInfo.ID]
			if ok {
				err := tx.Model(&blockInfo).Association("Locations").Delete(incoming.Locations)
				if err != nil {
					return fmt.Errorf("failed to append block info locations: %w", err)
				}
			}

			delete(incomingIndex, blockInfo.ID)
		}

		err = f.validateBlockInfos(tx, &fileInfo)
		if err != nil {
			return fmt.Errorf("failed to validate block info: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to upsert block info for file %s: %w", path, err)
	}

	return nil
}
