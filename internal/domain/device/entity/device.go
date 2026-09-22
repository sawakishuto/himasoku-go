package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Device struct {
	ID          uuid.UUID
	DeviceID    string
	UserID      uuid.UUID
	Platform    string
	PushToken   string
	Environment string
	// nil が「生きている端末」。APNs から BadDeviceToken を受けたときだけ
	// 打刻する。ゼロ値を持てる time.Time だと未打刻と区別できない。
	RevokedAt *time.Time
}

func NewDevice(deviceID, platform, pushToken, environment string) (*Device, error) {
	// RevokedAt は設定しない。登録した直後の端末は生きている。
	device := &Device{
		DeviceID:    deviceID,
		Platform:    platform,
		PushToken:   pushToken,
		Environment: environment,
	}

	if err := device.validate(); err != nil {
		return nil, err
	}

	return device, nil
}

func (d *Device) validate() error {
	if d.DeviceID == "" {
		return errors.New("deviceID is required")
	}
	if d.Platform == "" {
		return errors.New("platform is required")
	}
	if d.PushToken == "" {
		return errors.New("pushToken is required")
	}
	if d.Environment == "" {
		return errors.New("environment is required")
	}
	return nil
}

// 打刻の有無だけを見る。時刻の前後では判定しない。
// 送信対象を引く部分インデックスが revoked_at IS NULL を条件にしており、
// そこと食い違うと、インデックスから外れているのに生きている扱いになる。
func (d *Device) Is_revoked() (bool, error) {
	return d.RevokedAt != nil, nil
}
