package block

import (
	"fmt"
	"gorm.io/gorm"
	"strings"
)

type BlockInfo struct {
	ID           string `gorm:"primaryKey;uniqueIndex:idx_block_info;not null"`
	Sequence     uint64 `gorm:"not null;uniqueIndex:idx_block_info'"`
	Length       uint32 `gorm:"not null"`
	Path         string `gorm:"not null"`
	DataFilePath string `gorm:"not null"`
	CRC          uint32 `gorm:"not null"`
}

func (bi *BlockInfo) BeforeSave(_ *gorm.DB) error {
	bi.ID = strings.TrimSpace(bi.ID)
	bi.Path = strings.TrimSpace(bi.Path)
	bi.DataFilePath = strings.TrimSpace(bi.DataFilePath)

	if len(bi.ID) == 0 {
		return fmt.Errorf("block id is empty")
	}

	if len(bi.Path) == 0 {
		return fmt.Errorf("path is empty")
	}

	if len(bi.DataFilePath) == 0 {
		return fmt.Errorf("data file path is empty")
	}

	if bi.Length == 0 {
		return fmt.Errorf("length is zero")
	}

	return nil
}
