package ai

import (
	"context"
	"errors"
	"fmt"
	"myobj/src/internal/repository/impl"
	"myobj/src/pkg/models"

	"gorm.io/gorm"
)

func resolveUserFileInfo(ctx context.Context, factory *impl.RepositoryFactory, userID, fileID string) (*models.FileInfo, error) {
	if fileID == "" {
		return nil, fmt.Errorf("fileID is required")
	}

	if userFile, err := factory.UserFiles().GetByUserIDAndUfID(ctx, userID, fileID); err == nil {
		return factory.FileInfo().GetByID(ctx, userFile.FileID)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if userFile, err := factory.UserFiles().GetByUserIDAndFileID(ctx, userID, fileID); err == nil {
		return factory.FileInfo().GetByID(ctx, userFile.FileID)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return nil, gorm.ErrRecordNotFound
}
