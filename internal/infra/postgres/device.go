package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sawakishuto/himasoku-go/internal/dbgen"
	"github.com/sawakishuto/himasoku-go/internal/domain/device/entity"
	"github.com/sawakishuto/himasoku-go/internal/domain/device/repository"
)

var _ repository.DeviceRepository = (*DeviceRepository)(nil)

type DeviceRepository struct {
	pool *pgxpool.Pool
}

func NewDeviceRepository(pool *pgxpool.Pool) *DeviceRepository {
	return &DeviceRepository{pool: pool}
}

// Create は register_device() で upsert する。
//
// 直接 INSERT せず関数を使うのは、他人の push_token 行の所有者付け替えが
// RLS では書けないため。user_id は関数内で app_current_user_id() から取る。
func (r *DeviceRepository) Create(ctx context.Context, device *entity.Device) error {
	if device.UserID == uuid.Nil {
		return fmt.Errorf("device user id is required for RLS")
	}

	return runAsUser(ctx, r.pool, device.UserID, func(ctx context.Context, q *dbgen.Queries) error {
		row, err := q.RegisterDevice(ctx, dbgen.RegisterDeviceParams{
			Platform:        device.Platform,
			PushToken:       device.PushToken,
			ApnsEnvironment: nullableString(device.Environment),
		})
		if err != nil {
			return fmt.Errorf("failed to register device: %w", err)
		}
		applyDeviceRow(device, row)
		return nil
	})
}

func (r *DeviceRepository) FindByDeviceID(ctx context.Context, userID uuid.UUID, deviceID string) (*entity.Device, error) {
	var device *entity.Device
	err := runAsUser(ctx, r.pool, userID, func(ctx context.Context, q *dbgen.Queries) error {
		row, err := q.GetDeviceByPushToken(ctx, deviceID)
		if err != nil {
			return wrapDeviceNotFound(err, "failed to get device by push token")
		}
		device = toDeviceEntity(row)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return device, nil
}

func (r *DeviceRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Device, error) {
	var devices []*entity.Device
	err := runAsUser(ctx, r.pool, userID, func(ctx context.Context, q *dbgen.Queries) error {
		rows, err := q.ListDevicesByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
		if err != nil {
			return fmt.Errorf("failed to list devices for user %s: %w", userID, err)
		}
		devices = make([]*entity.Device, len(rows))
		for i, row := range rows {
			devices[i] = toDeviceEntity(row)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return devices, nil
}

func (r *DeviceRepository) Update(ctx context.Context, userID uuid.UUID, device *entity.Device) error {
	return runAsUser(ctx, r.pool, userID, func(ctx context.Context, q *dbgen.Queries) error {
		row, err := q.UpdateDevice(ctx, dbgen.UpdateDeviceParams{
			ID:              pgtype.UUID{Bytes: device.ID, Valid: true},
			Platform:        device.Platform,
			ApnsEnvironment: nullableString(device.Environment),
			RevokedAt:       timestamptzFromPtr(device.RevokedAt),
		})
		if err != nil {
			return wrapDeviceNotFound(err, fmt.Sprintf("failed to update device %s", device.ID))
		}
		applyDeviceRow(device, row)
		return nil
	})
}

func (r *DeviceRepository) Delete(ctx context.Context, userID uuid.UUID, deviceID string) error {
	return runAsUser(ctx, r.pool, userID, func(ctx context.Context, q *dbgen.Queries) error {
		if err := q.DeleteDevice(ctx, deviceID); err != nil {
			return fmt.Errorf("failed to delete device: %w", err)
		}
		return nil
	})
}

func toDeviceEntity(row dbgen.Device) *entity.Device {
	device := &entity.Device{
		ID:        uuid.UUID(row.ID.Bytes),
		DeviceID:  row.PushToken,
		UserID:    uuid.UUID(row.UserID.Bytes),
		Platform:  row.Platform,
		PushToken: row.PushToken,
	}
	if row.ApnsEnvironment != nil {
		device.Environment = *row.ApnsEnvironment
	}
	if row.RevokedAt.Valid {
		t := row.RevokedAt.Time
		device.RevokedAt = &t
	}
	return device
}

func applyDeviceRow(device *entity.Device, row dbgen.Device) {
	updated := toDeviceEntity(row)
	device.ID = updated.ID
	device.DeviceID = updated.DeviceID
	device.UserID = updated.UserID
	device.Platform = updated.Platform
	device.PushToken = updated.PushToken
	device.Environment = updated.Environment
	device.RevokedAt = updated.RevokedAt
}

func wrapDeviceNotFound(err error, msg string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: %w", msg, repository.ErrDeviceNotFound)
	}
	return fmt.Errorf("%s: %w", msg, err)
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func timestamptzFromPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}
