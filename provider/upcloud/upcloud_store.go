//go:build upcloud || !onlyprovider

package upcloud

import (
	"context"
	"math"
	"os"
	"time"

	"github.com/nanovms/ops/lepton"

	"github.com/UpCloudLtd/upcloud-go-api/v6/upcloud"
	"github.com/UpCloudLtd/upcloud-go-api/v6/upcloud/request"
)

func (p *Provider) createStorage(ctx *lepton.Context, storageName, filePath string) (storageDetails *upcloud.StorageDetails, err error) {
	fi, err := os.Stat(filePath)
	if err != nil {
		return
	}

	size := fi.Size()
	sizeGB := float64(size) * math.Pow(10, -9)

	rsz := int(math.Round(sizeGB*100) / 100)
	if rsz < 1 {
		rsz = 1
	}

	ctx.Logger().Info("creating empty storage")
	createReq := &request.CreateStorageRequest{
		Size:  rsz,
		Zone:  p.zone,
		Title: storageName,
	}

	storageDetails, err = p.upcloud.CreateStorage(context.Background(), createReq)
	if err != nil {
		return nil, err
	}

	ctx.Logger().Debug("%+v", storageDetails)

	ctx.Logger().Log("importing file system to storage, can take up to 5 minutes")
	importReq := &request.CreateStorageImportRequest{
		StorageUUID:    storageDetails.UUID,
		Source:         "direct_upload",
		SourceLocation: filePath,
	}

	importDetails, err := p.upcloud.CreateStorageImport(context.Background(), importReq)
	if err != nil {
		return nil, err
	}

	ctx.Logger().Debug("%+v", importDetails)

	err = p.waitForStorageState(storageDetails.UUID, "online")
	if err != nil {
		return nil, err
	}

	ctx.Logger().Info("import completed")

	return
}

func (p *Provider) waitForStorageState(uuid, state string) (err error) {
	waitStateReq := &request.WaitForStorageStateRequest{
		UUID:         uuid,
		DesiredState: state,
		Timeout:      10 * time.Minute,
	}

	_, err = p.upcloud.WaitForStorageState(context.Background(), waitStateReq)

	return
}

func (p *Provider) deleteStorage(uuid string) error {
	deleteStorageReq := &request.DeleteStorageRequest{
		UUID: uuid,
	}

	return p.upcloud.DeleteStorage(context.Background(), deleteStorageReq)
}
