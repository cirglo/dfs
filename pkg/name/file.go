package name

import (
	"database/sql"
	"fmt"
	"github.com/cirglo.com/dfs/pkg/proto"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sort"
	"strings"
	"time"
)

type BlockReport struct {
	ID       string
	Path     string
	Hosts    []string
	Sequence uint64
	Length   uint32
	CRC      uint32
}

type FileService interface {
	Stat(p Principal, path string) (FileInfo, error)
	List(p Principal, path string) ([]FileInfo, error)
	CreateFile(p Principal, path string, perms Permissions) (FileInfo, error)
	CreateDir(p Principal, path string, perms Permissions) (FileInfo, error)
	DeleteFile(p Principal, path string) error
	DeleteDir(p Principal, path string) error
	GetBlockInfos(p Principal, path string) ([]BlockInfo, error)
	NotifyBlockPresent(n *proto.NotifyBlockPresentRequest) error
	NotifyBlockAdded(n *proto.NotifyBlockAddedRequest) error
	NotifyBlockRemoved(n *proto.NotifyBlockRemovedRequest) error
}

type FileInfo struct {
	ID          uint64      `gorm:"autoIncrement;primaryKey"`
	CreatedAt   time.Time   `gorm:"autoCreateTime"`
	UpdatedAt   time.Time   `gorm:"autoUpdateTime"`
	ParentID    *uint64     `gorm:"uniqueIndex:idx_fileinfo_name;foreignKey:id"`
	Name        string      `gorm:"size:256;uniqueIndex:idx_fileinfo_name;not null"`
	IsDir       bool        `gorm:"not null"`
	Children    []FileInfo  `gorm:"foreignKey:ParentID"`
	Permissions Permissions `gorm:"embedded;embeddedPrefix:permissions_"`
	BlockInfos  []BlockInfo `gorm:"constraint:OnDelete:CASCADE"`
}

var _ HasPermissions = &FileInfo{}

func (fi *FileInfo) GetSize() uint64 {
	size := uint64(0)

	for _, blockInfo := range fi.BlockInfos {
		size += uint64(blockInfo.Length)
	}

	return size
}

func (fi *FileInfo) IsHealthy() bool {
	blockInfos := fi.BlockInfos

	sort.Slice(blockInfos, func(i, j int) bool {
		return blockInfos[i].Sequence < blockInfos[j].Sequence
	})

	for i, blockInfo := range blockInfos {
		if blockInfo.Sequence != uint64(i) {
			return false
		}

		if len(blockInfo.Locations) == 0 {
			return false
		}
	}

	return true

}

func (fi *FileInfo) BeforeSave(_ *gorm.DB) error {
	fi.Name = strings.TrimSpace(fi.Name)

	if strings.Contains(fi.Name, "/") {
		return fmt.Errorf("invalid name, cannot contain '/': '%s'", fi.Name)
	}

	if fi.IsDir {
		if fi.ParentID == nil {
			if len(fi.Name) > 0 {
				return fmt.Errorf("root directory cannot have a name but was '%s'", fi.Name)
			}
		} else {
			if len(fi.Name) == 0 {
				return fmt.Errorf("directory must have a name")
			}
		}

		if len(fi.BlockInfos) > 0 {
			return fmt.Errorf("directory cannot have blocks")
		}
	} else {
		if len(fi.Name) == 0 {
			return fmt.Errorf("file must have a name")
		}

		if fi.ParentID == nil {
			return fmt.Errorf("file must have a parent")
		}

		if len(fi.Children) > 0 {
			return fmt.Errorf("file cannot have children")
		}
	}

	return nil
}

func (fi *FileInfo) AfterFind(tx *gorm.DB) error {
	blockInfos := fi.BlockInfos

	sort.Slice(blockInfos, func(i, j int) bool {
		return blockInfos[i].Sequence < blockInfos[j].Sequence
	})

	return nil
}

func (fi *FileInfo) FindChild(name string) (FileInfo, bool) {
	for _, child := range fi.Children {
		if child.Name == name {
			return child, true
		}
	}

	return FileInfo{}, false
}

func (fi *FileInfo) GetPermissions() Permissions {
	return fi.Permissions
}

type BlockInfo struct {
	ID         string     `gorm:"primaryKey;not null"`
	CreatedAt  time.Time  `gorm:"autoCreateTime"`
	UpdatedAt  time.Time  `gorm:"autoUpdateTime"`
	Locations  []Location `gorm:"constraint:OnDelete:CASCADE"`
	FileInfoID uint64     `gorm:"uniqueIndex:idx_blockinfo_sequence;not null"`
	Sequence   uint64     `gorm:"uniqueIndex:idx_blockinfo_sequence;not null"`
	Length     uint32     `gorm:"not null"`
	CRC        uint32     `gorm:"not null"`
}

func (bi *BlockInfo) BeforeSave(_ *gorm.DB) error {
	bi.ID = strings.TrimSpace(bi.ID)

	if len(bi.ID) == 0 {
		return fmt.Errorf("ID cannot be empty")
	}

	return nil
}

func (bi *BlockInfo) AfterFind(tx *gorm.DB) error {
	locations := bi.Locations

	sort.Slice(locations, func(i, j int) bool {
		return locations[i].Host < locations[j].Host
	})

	return nil
}

func (bi *BlockInfo) ContainsHost(host string) bool {
	for _, location := range bi.Locations {
		if location.Host == host {
			return true
		}
	}

	return false
}

type Location struct {
	BlockInfoID string `gorm:"uniqueIndex:idx_location;not null"`
	Host        string `gorm:"uniqueIndex:idx_location;not null"`
}

func (l *Location) BeforeSave(_ *gorm.DB) error {
	l.Host = strings.TrimSpace(l.Host)
	if len(l.Host) == 0 {
		return fmt.Errorf("location host is empty")
	}

	return nil
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

	fileService := fileService{
		Opts: opts,
	}

	//create root directory if it doesn't exist
	err := opts.DB.Transaction(func(tx *gorm.DB) error {
		rootDir, err := fileService.lookupRoot(tx)
		if err != nil {
			opts.Logger.WithError(err).Info("Could not find root directory. Creating...")
			rootDir.IsDir = true
			rootDir.ParentID = nil
			rootDir.Permissions = Permissions{
				Owner: "root",
				Group: "root",
				OwnerPermission: Permission{
					Read:   true,
					Write:  true,
					Delete: true,
				},
				GroupPermission: Permission{
					Read:   true,
					Write:  true,
					Delete: true,
				},
				OtherPermission: Permission{
					Read:   true,
					Write:  true,
					Delete: true,
				},
			}

			err = tx.Create(&rootDir).Error
			if err != nil {
				return fmt.Errorf("could not create root diretory: %w", err)
			}

			opts.Logger.WithField("file-info", rootDir).Info("Created root directory")
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not initialize file system: %w", err)
	}

	return &fileService, nil
}

func (f *fileService) cleanPath(path string) (string, string, string, error) {
	trimmedPath := strings.TrimSpace(path)
	trimmedPath = strings.TrimSuffix(trimmedPath, "/")

	if len(trimmedPath) == 0 {
		return "", "", "", fmt.Errorf("path cannot be empty")
	}

	if !strings.HasPrefix(trimmedPath, "/") {
		return "", "", "", fmt.Errorf("path must start with / but was '%s'", trimmedPath)
	}

	if trimmedPath == "/" {
		return "/", "/", "", nil
	}

	if strings.Count(trimmedPath, "/") == 1 {
		name := strings.TrimPrefix(trimmedPath, "/")
		return trimmedPath, "/", name, nil
	}

	lastIndex := strings.LastIndex(trimmedPath, "/")
	if lastIndex == -1 {
		return "", "", "", fmt.Errorf("invalid path '%s", trimmedPath)
	}

	parentPath := trimmedPath[0:lastIndex]
	name := trimmedPath[lastIndex+1:]

	return trimmedPath, parentPath, name, nil
}

func (f *fileService) lookupRoot(tx *gorm.DB) (FileInfo, error) {
	fileInfo := FileInfo{}
	err := tx.Where(&FileInfo{ParentID: nil}, "ParentID").Preload(clause.Associations).First(&fileInfo).Error
	if err != nil {
		return fileInfo, fmt.Errorf("could not lookup root directory")
	}

	if !fileInfo.IsDir {
		return fileInfo, fmt.Errorf("root directory is not a directory")
	}

	return fileInfo, nil
}

func (f *fileService) lookup(tx *gorm.DB, path string) ([]FileInfo, error) {
	var fileInfos []FileInfo

	rootDir, err := f.lookupRoot(tx)
	if err != nil {
		return nil, fmt.Errorf("could not lookup root directory: %w", err)
	}

	fileInfos = append(fileInfos, rootDir)

	if strings.TrimSpace(path) == "/" {
		return fileInfos, nil
	}

	cleanPath, _, _, err := f.cleanPath(path)
	if err != nil {
		return nil, fmt.Errorf("could not clean path: %w", err)
	}

	cleanPath = strings.TrimPrefix(cleanPath, "/")
	parts := strings.Split(cleanPath, "/")
	f.Opts.Logger.WithFields(logrus.Fields{
		"path":       path,
		"clean-path": cleanPath,
		"parts":      parts,
	}).Debug("Cleaned path")

	currentDir := rootDir

	for _, part := range parts {
		child, found := currentDir.FindChild(part)
		if !found {
			return []FileInfo{},
				fmt.Errorf(
					"could not find child '%s' of path '%s' in %v",
					part,
					path,
					currentDir)
		}

		fileInfos = append(fileInfos, child)
		currentDir = child
		err := tx.Model(&currentDir).Preload(clause.Associations).Error
		if err != nil {
			return []FileInfo{}, fmt.Errorf("could not preload child %v", currentDir)
		}

		if !currentDir.IsDir {
			break
		}
	}

	return fileInfos, nil
}

func (f *fileService) computePrivileges(p Principal, fileInfos ...FileInfo) Privileges {
	hasPermissionsList := []HasPermissions{}

	for _, fileInfo := range fileInfos {
		hasPermissionsList = append(hasPermissionsList, &fileInfo)
	}

	return p.ComputePrivileges(hasPermissionsList...)
}

func (f *fileService) canRead(p Principal, fileInfos ...FileInfo) bool {
	return f.computePrivileges(p, fileInfos...).Read
}

func (f *fileService) canWrite(p Principal, fileInfos ...FileInfo) bool {
	return f.computePrivileges(p, fileInfos...).Write
}

func (f *fileService) canDelete(p Principal, fileInfos ...FileInfo) bool {
	return f.computePrivileges(p, fileInfos...).Delete
}

func (f *fileService) Stat(p Principal, path string) (FileInfo, error) {
	var fileInfo FileInfo
	err := f.Opts.DB.Transaction(func(tx *gorm.DB) error {
		fileInfos, err := f.lookup(tx, path)
		if err != nil {
			return fmt.Errorf("failed to lookup %s: %w", path, err)
		}

		if !f.canRead(p, fileInfos...) {
			return fmt.Errorf("permission denied for %s", path)
		}

		fileInfo = fileInfos[len(fileInfos)-1]

		return nil
	}, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to stat %s: %w", path, err)
	}

	return fileInfo, nil
}

func (f *fileService) List(p Principal, path string) ([]FileInfo, error) {
	var children []FileInfo
	err := f.Opts.DB.Transaction(func(tx *gorm.DB) error {
		fileInfos, err := f.lookup(tx, path)
		if err != nil {
			return fmt.Errorf("failed to lookup dirs: %w", err)
		}

		if !f.canRead(p, fileInfos...) {
			return fmt.Errorf("permission denied for %s", path)
		}

		target := fileInfos[len(fileInfos)-1]
		children = target.Children

		return nil
	}, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	sort.Slice(children, func(i, j int) bool {
		return children[i].Name < children[j].Name
	})

	return children, nil
}

func (f *fileService) CreateFile(p Principal, path string, perms Permissions) (FileInfo, error) {
	var fileInfo FileInfo

	_, parentPath, name, err := f.cleanPath(path)
	if err != nil {
		return FileInfo{}, fmt.Errorf("invalid path '%s': %w", path, err)
	}

	err = f.Opts.DB.Transaction(func(tx *gorm.DB) error {
		parents, err := f.lookup(tx, parentPath)
		if err != nil {
			return fmt.Errorf("failed to lookup parent: %w", err)
		}

		if len(parents) == 0 {
			return fmt.Errorf("no parent found")
		}

		parent := parents[len(parents)-1]

		if !f.canWrite(p, parent) {
			return fmt.Errorf("permission denied")
		}

		fileInfo = FileInfo{
			Name:        name,
			IsDir:       false,
			ParentID:    &parent.ID,
			Permissions: perms,
		}

		err = tx.Create(&fileInfo).Error
		if err != nil {
			return fmt.Errorf("failed to create: %w", err)
		}

		return nil
	})
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to create file '%s': %w", path, err)
	}

	return fileInfo, nil
}

func (f *fileService) CreateDir(p Principal, path string, perms Permissions) (FileInfo, error) {
	var fileInfo FileInfo

	_, parentPath, name, err := f.cleanPath(path)
	if err != nil {
		return FileInfo{}, fmt.Errorf("invalid path '%s': %w", path, err)
	}

	err = f.Opts.DB.Transaction(func(tx *gorm.DB) error {
		parents, err := f.lookup(tx, parentPath)
		if err != nil {
			return fmt.Errorf("failed to lookup parent: %w", err)
		}

		if len(parents) == 0 {
			return fmt.Errorf("no parent found")
		}

		parent := parents[len(parents)-1]

		if !f.canWrite(p, parent) {
			return fmt.Errorf("permission denied")
		}

		fileInfo = FileInfo{
			Name:        name,
			IsDir:       true,
			ParentID:    &parent.ID,
			Permissions: perms,
		}

		err = tx.Create(&fileInfo).Error
		if err != nil {
			return fmt.Errorf("failed to create: %w", err)
		}

		return nil
	})
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to create dir '%s': %w", path, err)
	}

	return fileInfo, nil
}

func (f *fileService) DeleteFile(p Principal, path string) error {
	err := f.Opts.DB.Transaction(func(tx *gorm.DB) error {
		fileInfos, err := f.lookup(tx, path)
		if err != nil {
			return fmt.Errorf("failed to lookup file: %w", err)
		}

		if !f.canDelete(p, fileInfos...) {
			return fmt.Errorf("permission denied")
		}

		if len(fileInfos) == 0 {
			return fmt.Errorf("no file found")
		}

		fileInfo := fileInfos[len(fileInfos)-1]

		if fileInfo.IsDir {
			return fmt.Errorf("can't delete a directory with this call")
		}

		err = tx.Delete(&fileInfo).Error
		if err != nil {
			return fmt.Errorf("failed to delete file: %w", err)
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
		fileInfos, err := f.lookup(tx, path)
		if err != nil {
			return fmt.Errorf("failed to lookup directory: %w", err)
		}

		if !f.canDelete(p, fileInfos...) {
			return fmt.Errorf("permission denied")
		}

		if len(fileInfos) == 0 {
			return fmt.Errorf("no directory found")
		}

		fileInfo := fileInfos[len(fileInfos)-1]

		if !fileInfo.IsDir {
			return fmt.Errorf("can't delete a file with this call")
		}

		if len(fileInfo.Children) > 0 {
			return fmt.Errorf("directory is not empty")
		}
		err = tx.Delete(&fileInfo).Error
		if err != nil {
			return fmt.Errorf("failed to delete: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to delete directory %s: %w", path, err)
	}

	return nil
}

func (f *fileService) GetBlockInfos(p Principal, path string) ([]BlockInfo, error) {
	var blockInfos []BlockInfo
	err := f.Opts.DB.Transaction(func(tx *gorm.DB) error {
		fileInfos, err := f.lookup(tx, path)
		if err != nil {
			return fmt.Errorf("failed to lookup file: %w", err)
		}

		if !f.canRead(p, fileInfos...) {
			return fmt.Errorf("permission denied")
		}

		if len(fileInfos) == 0 {
			return fmt.Errorf("no file found")
		}

		blockInfos = fileInfos[len(fileInfos)-1].BlockInfos

		return nil
	}, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get block infos for file %s: %w", path, err)
	}

	return blockInfos, nil
}

func (f *fileService) NotifyBlockPresent(n *proto.NotifyBlockPresentRequest) error {
	err := f.Opts.DB.Transaction(func(tx *gorm.DB) error {
		blockInfo := BlockInfo{}

		fileInfos, err := f.lookup(tx, n.Path)
		if err != nil {
			return fmt.Errorf("file not found: %w", err)
		}

		if len(fileInfos) == 0 {
			return fmt.Errorf("file not found")
		}

		fileInfo := fileInfos[len(fileInfos)-1]

		if fileInfo.IsDir {
			return fmt.Errorf("path is a directory")
		}

		err = tx.Where(
			&BlockInfo{ID: n.GetBlockId()}).
			Attrs(&BlockInfo{
				ID:         n.GetBlockId(),
				FileInfoID: fileInfo.ID,
				Sequence:   n.GetSequence(),
				Length:     n.GetLength(),
				CRC:        n.GetCrc(),
			}).FirstOrCreate(&blockInfo).Error
		if err != nil {
			return fmt.Errorf("could not get or create block: %w", err)
		}

		if blockInfo.Sequence != n.GetSequence() {
			return fmt.Errorf("sequence %d does not match %d", n.GetSequence(), blockInfo.Sequence)
		}

		if blockInfo.Length != n.GetLength() {
			return fmt.Errorf("length %d does not match %d", n.GetLength(), blockInfo.Length)
		}

		if blockInfo.CRC != n.GetCrc() {
			return fmt.Errorf("crc %d does not match %d", n.GetCrc(), blockInfo.CRC)
		}

		if !blockInfo.ContainsHost(n.GetHost()) {
			location := Location{
				BlockInfoID: blockInfo.ID,
				Host:        n.Host,
			}

			err = tx.Create(&location).Error
			if err != nil {
				return fmt.Errorf("could not create location: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf(
			"failed to notify block '%s' exists for path '%s' at host '%s': %w",
			n.GetBlockId(),
			n.GetPath(),
			n.GetHost(),
			err)
	}

	return nil
}

func (f *fileService) NotifyBlockAdded(n *proto.NotifyBlockAddedRequest) error {
	err := f.Opts.DB.Transaction(func(tx *gorm.DB) error {
		blockInfo := BlockInfo{}

		fileInfos, err := f.lookup(tx, n.Path)
		if err != nil {
			return fmt.Errorf("file not found: %w", err)
		}

		if len(fileInfos) == 0 {
			return fmt.Errorf("file not found")
		}

		fileInfo := fileInfos[len(fileInfos)-1]

		if fileInfo.IsDir {
			return fmt.Errorf("path is a directory")
		}

		err = tx.Where(
			&BlockInfo{ID: n.GetBlockId()}).
			Attrs(&BlockInfo{
				ID:         n.GetBlockId(),
				FileInfoID: fileInfo.ID,
				Sequence:   n.GetSequence(),
				Length:     n.GetLength(),
				CRC:        n.GetCrc(),
			}).FirstOrCreate(&blockInfo).Error
		if err != nil {
			return fmt.Errorf("could not get or create block: %w", err)
		}

		if blockInfo.Sequence != n.GetSequence() {
			return fmt.Errorf("sequence %d does not match %d", n.GetSequence(), blockInfo.Sequence)
		}

		if blockInfo.Length != n.GetLength() {
			return fmt.Errorf("length %d does not match %d", n.GetLength(), blockInfo.Length)
		}

		if blockInfo.CRC != n.GetCrc() {
			return fmt.Errorf("crc %d does not match %d", n.GetCrc(), blockInfo.CRC)
		}

		location := Location{
			BlockInfoID: blockInfo.ID,
			Host:        n.Host,
		}

		err = tx.Create(&location).Error
		if err != nil {
			return fmt.Errorf("could not create location: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf(
			"failed to notify block '%s' added for path '%s' at host '%s': %w",
			n.GetBlockId(),
			n.GetPath(),
			n.GetHost(),
			err)
	}

	return nil
}
func (f *fileService) NotifyBlockRemoved(n *proto.NotifyBlockRemovedRequest) error {
	err := f.Opts.DB.Transaction(func(tx *gorm.DB) error {
		blockInfo := BlockInfo{}

		fileInfos, err := f.lookup(tx, n.Path)
		if err != nil {
			return fmt.Errorf("file not found: %w", err)
		}

		if len(fileInfos) == 0 {
			return fmt.Errorf("file not found")
		}

		fileInfo := fileInfos[len(fileInfos)-1]

		if fileInfo.IsDir {
			return fmt.Errorf("path is a directory")
		}

		err = tx.Where(
			&BlockInfo{ID: n.GetBlockId()}).First(&blockInfo).Error
		if err != nil {
			return fmt.Errorf("could not get block: %w", err)
		}

		for _, location := range blockInfo.Locations {
			if location.Host == n.GetHost() {
				err = tx.Delete(&location).Error
				if err != nil {
					return fmt.Errorf("could not delete location")
				}
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf(
			"failed to notify block '%s' at path '%s' removed from host '%s': %w",
			n.GetBlockId(),
			n.GetPath(),
			n.GetHost(),
			err)
	}

	return nil
}
