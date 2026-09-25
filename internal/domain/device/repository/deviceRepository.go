package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/sawakishuto/himasoku-go/internal/domain/device/entity"
)

// ErrDeviceNotFound は該当する端末が存在しない、または RLS により見えないことを表す。
var ErrDeviceNotFound = errors.New("device not found")

type DeviceRepository interface {
	// Create は端末を登録する。device.UserID は RLS の実行主体として使われる。
	Create(ctx context.Context, device *entity.Device) error
	// FindByDeviceID は push_token（entity.DeviceID）で引く。userID は RLS 用。
	FindByDeviceID(ctx context.Context, userID uuid.UUID, deviceID string) (*entity.Device, error)
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Device, error)
	Update(ctx context.Context, userID uuid.UUID, device *entity.Device) error
	// Delete は行を物理削除する。履歴を残す失効は Update で revoked_at を打刻する。
	Delete(ctx context.Context, userID uuid.UUID, deviceID string) error
}
