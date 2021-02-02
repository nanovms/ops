package upcloud

import (
	"math"
	"os"
	"time"

	"github.com/UpCloudLtd/upcloud-go-api/upcloud"
	"github.com/UpCloudLtd/upcloud-go-api/upcloud/request"
	"github.com/nanovms/ops/lepton"
)

func (p *Provider) createStorage(ctx *lepton.Context, storageName, filePath string) (storageDetails *upcloud.StorageDetails, err error) {
	fi, err := os.Stat(filePath)
	if err != nil {
		return
	}

	size := fi.Size()
	sizeGB := float64(size) * math.Pow(10, -9)

	if sizeGB < 10 {
		sizeGB = 10
	} else if sizeGB > 2048 {
		sizeGB = 2048
	}

	ctx.Logger().Info("creating empty storage")
	createReq := &request.CreateStorageRequest{
		Size:  int(sizeGB),
		Zone:  p.zone,
		Title: storageName,
	}

	storageDetails, err = p.upcloud.CreateStorage(createReq)
	if err != nil {
		return
	}

	ctx.Logger().Debug("%+v", storageDetails)

	ctx.Logger().Log("importing file system to storage, can take up to 5 minutes")
	importReq := &request.CreateStorageImportRequest{
		StorageUUID:    storageDetails.UUID,
		Source:         "direct_upload",
		SourceLocation: filePath,
	}

	importDetails, err := p.upcloud.CreateStorageImport(importReq)
	if err != nil {
		return
	}

	ctx.Logger().Debug("%+v", importDetails)

	err = p.waitForStorageState(storageDetails.UUID, "online")
	if err != nil {
		return
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

	_, err = p.upcloud.WaitForStorageState(waitStateReq)

	return
}

func (p *Provider) deleteStorage(uuid string) error {
	deleteStorageReq := &request.DeleteStorageRequest{
		UUID: uuid,
	}

	return p.upcloud.DeleteStorage(deleteStorageReq)
}
