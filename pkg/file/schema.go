package file

import (
	"fmt"
	"github.com/cirglo.com/dfs/pkg/security"
	"gorm.io/gorm"
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

type FileInfo struct {
	ID          uint64               `gorm:"autoIncrement;primaryKey"`
	CreatedAt   time.Time            `gorm:"autoCreateTime"`
	UpdatedAt   time.Time            `gorm:"autoUpdateTime"`
	ParentID    *uint64              `gorm:"uniqueIndex:idx_fileinfo_name;foreignKey:id"`
	Name        string               `gorm:"size:256;uniqueIndex:idx_fileinfo_name;not null"`
	IsDir       bool                 `gorm:"not null"`
	Children    []FileInfo           `gorm:"foreignKey:ParentID"`
	Permissions security.Permissions `gorm:"embedded;embeddedPrefix:permissions_"`
	BlockInfos  []BlockInfo          `gorm:"constraint:OnDelete:CASCADE"`
}

var _ security.HasPermissions = &FileInfo{}

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

func (fi *FileInfo) GetPermissions() security.Permissions {
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
