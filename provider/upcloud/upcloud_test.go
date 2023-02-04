package upcloud_test

import (
	"errors"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/mock_provider/mock_upcloud"
	"github.com/nanovms/ops/provider/upcloud"
	"github.com/stretchr/testify/assert"
)

func TestProviderInitialize(t *testing.T) {

	t.Run("should return error if upcloud user is undefined", func(t *testing.T) {
		p, s := NewProvider(t)

		s.EXPECT().
			GetAccount().
			Return(nil, nil)

		err := p.Initialize(&lepton.NewConfig().CloudConfig)

		assert.Error(t, err)
	})

	t.Run("should return error if upcloud password is undefined", func(t *testing.T) {
		os.Setenv("UPCLOUD_USER", "test")

		p, s := NewProvider(t)

		s.EXPECT().
			GetAccount().
			Return(nil, nil)

		err := p.Initialize(&lepton.NewConfig().CloudConfig)

		assert.Error(t, err)
	})

	t.Run("should return error if upcloud zone is undefined", func(t *testing.T) {
		os.Setenv("UPCLOUD_USER", "test")
		os.Setenv("UPCLOUD_PASSWORD", "password")

		p, s := NewProvider(t)

		s.EXPECT().
			GetAccount().
			Return(nil, nil)

		err := p.Initialize(&lepton.NewConfig().CloudConfig)

		assert.Error(t, err)
	})

	t.Run("should return error if client credentials are invalid", func(t *testing.T) {
		os.Setenv("UPCLOUD_USER", "test")
		os.Setenv("UPCLOUD_PASSWORD", "password")
		os.Setenv("UPCLOUD_ZONE", "us")

		p, s := NewProvider(t)

		errInvalidCredentials := errors.New("invalid credentials")

		s.EXPECT().
			GetAccount().
			Return(nil, errInvalidCredentials)

		err := p.Initialize(&lepton.NewConfig().CloudConfig)

		assert.Error(t, err)
	})

	t.Run("should not throw error if credentials are valid", func(t *testing.T) {
		os.Setenv("UPCLOUD_USER", "test")
		os.Setenv("UPCLOUD_PASSWORD", "password")
		os.Setenv("UPCLOUD_ZONE", "us")

		p, s := NewProvider(t)

		s.EXPECT().
			GetAccount().
			Return(nil, nil)

		err := p.Initialize(&lepton.NewConfig().CloudConfig)

		assert.Nil(t, err)
	})
}

func NewProvider(t *testing.T) (*upcloud.Provider, *mock_upcloud.MockService) {
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	service := mock_upcloud.NewMockService(ctrl)

	return upcloud.NewProviderWithService(service), service
}
