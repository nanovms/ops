package hetzner

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/nanovms/ops/lepton"
	"github.com/nanovms/ops/log"
	"github.com/nanovms/ops/types"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/crypto/ssh"
)

const (
	defaultTargetDevice = "/dev/sda"
	cloudInitTemplate   = `#cloud-config
write_files:
  - path: /tmp/write-os-image.sh
    permissions: '0755'
    content: |
      #!/bin/bash
      set -euo pipefail

      IMAGE_URL="{{ .ImageURL }}"
      IMAGE_PATH="/dev/shm/disk.img"
      TARGET_DEVICE="{{ .TargetDevice }}"
      BLOCK_SIZE="4M"

      curl --fail --location --progress-bar -o "$IMAGE_PATH" "$IMAGE_URL"

      cp /usr/bin/dd /dev/shm/dd
      blkdiscard "$TARGET_DEVICE" -f || true
      /dev/shm/dd if="$IMAGE_PATH" of="$TARGET_DEVICE" bs="$BLOCK_SIZE" conv=sync

      # Sync and shut down
      echo s > /proc/sysrq-trigger
      echo o > /proc/sysrq-trigger
runcmd:
  - /tmp/write-os-image.sh`
)

type cloudInitData struct {
	ImageURL     string
	TargetDevice string
}

// CustomizeImage returns image path with adaptations needed by cloud provider
func (h *Hetzner) CustomizeImage(ctx *lepton.Context) (string, error) {
	imagePath := ctx.Config().RunConfig.ImageName
	return imagePath, nil
}

func (h *Hetzner) BuildImage(ctx *lepton.Context) (string, error) {
	c := ctx.Config()
	if !c.Uefi {
		return "", fmt.Errorf("hetzner images require 'Uefi; true' it in your configuration")
	}
	if err := lepton.BuildImage(*c); err != nil {
		return "", err
	}

	return h.CustomizeImage(ctx)
}

func (h *Hetzner) BuildImageWithPackage(ctx *lepton.Context, pkgpath string) (string, error) {
	c := ctx.Config()
	if !c.Uefi {
		return "", fmt.Errorf("hetzner images require 'Uefi; true' it in your configuration")
	}
	if err := lepton.BuildImageFromPackage(pkgpath, *c); err != nil {
		return "", err
	}
	return h.CustomizeImage(ctx)
}

func (h *Hetzner) CreateImage(ctx *lepton.Context, imagePath string) error {
	config := ctx.Config()
	logger := ctx.Logger()
	h.ensureStorage()

	if imagePath == "" {
		opshome := lepton.GetOpsHome()
		imagePath = filepath.Join(opshome, "images", config.CloudConfig.ImageName)
	}

	if err := h.Storage.CopyToBucket(config, imagePath); err != nil {
		return err
	}

	objectKey := filepath.Base(imagePath)
	publicURL := h.Storage.getImageObjectStorageURL(config, objectKey)
	if publicURL == "" {
		return errors.New("hetzner object storage url could not be derived; check bucket/zone configuration")
	}

	logger.Infof("uploaded image to %s", publicURL)

	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	serverTypeName := strings.TrimSpace(config.CloudConfig.Flavor)
	if serverTypeName == "" {
		serverTypeName = defaultServerType
	}

	serverType, _, err := h.Client.ServerType.GetByName(ctxWithTimeout, serverTypeName)
	if err != nil {
		return err
	}
	if serverType == nil {
		return fmt.Errorf("unable to resolve server type %q", serverTypeName)
	}

	baseImageRef := strings.TrimSpace(config.CloudConfig.ImageType)
	if baseImageRef == "" {
		baseImageRef = defaultBuilderImage
	}

	baseImage, err := h.resolveImageReference(ctxWithTimeout, baseImageRef)
	if err != nil {
		return err
	}
	if baseImage == nil {
		return fmt.Errorf("unable to resolve builder image %q", baseImageRef)
	}

	userData, err := renderCloudConfig(publicURL, defaultTargetDevice)
	if err != nil {
		return err
	}

	var sshKey *hcloud.SSHKey
	if sshKey, err = h.createEphemeralSSHKey(ctxWithTimeout); err != nil {
		return err
	}
	defer func() {
		_, err := h.Client.SSHKey.Delete(context.Background(), sshKey)
		if err != nil {
			logger.Warnf("failed to delete temporary ssh key: %v", err)
		}
	}()

	zone := strings.TrimSpace(config.CloudConfig.Zone)

	builderServer, err := h.createServer(ctxWithTimeout, serverCreateParams{
		Name:       fmt.Sprintf("ops-builder-%d", time.Now().UnixNano()),
		ServerType: serverType,
		Image:      baseImage,
		UserData:   userData,
		SSHKeys:    []*hcloud.SSHKey{sshKey},
		BaseLabels: map[string]string{
			opsImageBuilderLabel: "true",
		},
		Tags:         config.CloudConfig.Tags,
		TagFilter:    func(tag types.Tag) bool { return tag.IsInstanceLabel() },
		LocationName: zone,
	})
	if err != nil {
		return err
	}

	defer func() {
		if _, _, derr := h.Client.Server.DeleteWithResult(context.Background(), builderServer); derr != nil {
			logger.Warnf("failed to delete builder server %q: %v", builderServer.Name, derr)
		}
	}()

	if err := waitForServerStatus(ctxWithTimeout, h.Client, builderServer.ID, hcloud.ServerStatusOff); err != nil {
		return err
	}

	imageLabels := combineLabels(map[string]string{
		opsLabelKey:          opsLabelValue,
		opsImageNameLabelKey: config.CloudConfig.ImageName,
	}, config.CloudConfig.Tags, func(tag types.Tag) bool { return tag.IsImageLabel() })

	description := config.Description
	if description == "" {
		description = config.CloudConfig.ImageName
	}

	createImage, _, err := h.Client.Server.CreateImage(ctxWithTimeout, builderServer, &hcloud.ServerCreateImageOpts{
		Type:        hcloud.ImageTypeSnapshot,
		Description: &description,
		Labels:      imageLabels,
	})
	if err != nil {
		return err
	}

	if createImage.Action != nil {
		if err := h.Client.Action.WaitFor(ctxWithTimeout, createImage.Action); err != nil {
			return err
		}
	}

	logger.Infof("snapshot %q (%d) created", createImage.Image.Name, createImage.Image.ID)

	if err := h.Storage.DeleteFromBucket(config, objectKey); err != nil {
		logger.Warnf("failed to delete object %q from bucket: %v", objectKey, err)
	}

	return nil
}

func (h *Hetzner) ListImages(ctx *lepton.Context, filter string) error {
	images, err := h.GetImages(ctx, filter)
	if err != nil {
		return err
	}

	if ctx.Config().RunConfig.JSON {
		return jsonOutput(images)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "ID", "Status", "Size", "Created"})
	table.SetRowLine(true)

	for _, image := range images {
		size := ""
		if image.Size > 0 {
			size = lepton.Bytes2Human(image.Size)
		}
		row := []string{
			image.Name,
			image.ID,
			image.Status,
			size,
			lepton.Time2Human(image.Created),
		}
		table.Append(row)
	}

	table.Render()
	return nil
}

func (h *Hetzner) GetImages(ctx *lepton.Context, filter string) ([]lepton.CloudImage, error) {
	selector := managedLabelSelector()
	if strings.TrimSpace(filter) != "" {
		selector = fmt.Sprintf("%s,%s=%s", selector, opsImageNameLabelKey, filter)
	}

	opts := hcloud.ImageListOpts{
		ListOpts: hcloud.ListOpts{
			LabelSelector: selector,
		},
		Type: []hcloud.ImageType{hcloud.ImageTypeSnapshot},
	}

	images, err := h.Client.Image.AllWithOpts(context.Background(), opts)
	if err != nil {
		return nil, err
	}

	result := make([]lepton.CloudImage, 0, len(images))
	for _, img := range images {
		name := labelValue(img.Labels, opsImageNameLabelKey)
		if name == "" {
			name = img.Name
		}
		result = append(result, lepton.CloudImage{
			ID:      strconv.FormatInt(img.ID, 10),
			Name:    name,
			Status:  string(img.Status),
			Size:    int64(img.DiskSize * lepton.GB),
			Created: img.Created,
			Labels:  flattenLabels(img.Labels),
		})
	}

	return result, nil
}

func (h *Hetzner) DeleteImage(ctx *lepton.Context, imagename string) error {
	config := ctx.Config()
	image, err := h.fetchSnapshotByName(context.Background(), imagename)
	if err != nil {
		return err
	}
	if image == nil {
		return fmt.Errorf(`image with name "%s" not found`, imagename)
	}

	if _, err := h.Client.Image.Delete(context.Background(), image); err != nil {
		return err
	}

	h.ensureStorage()
	if err := h.Storage.DeleteFromBucket(config, filepath.Base(imagename)); err != nil {
		ctx.Logger().Warnf("failed to remove %q from bucket: %v", imagename, err)
	}

	return nil
}

func (*Hetzner) ResizeImage(ctx *lepton.Context, imagename string, hbytes string) error {
	return fmt.Errorf("operation not supported")
}

func (*Hetzner) SyncImage(config *types.Config, target lepton.Provider, imagename string) error {
	log.Warn("not yet implemented")
	return nil
}

func renderCloudConfig(imageURL string, targetDevice string) (string, error) {
	if targetDevice == "" {
		targetDevice = defaultTargetDevice
	}

	tmpl, err := template.New("cloud-config").Parse(cloudInitTemplate)
	if err != nil {
		return "", fmt.Errorf("error parsing template: %w", err)
	}

	var buf bytes.Buffer
	data := cloudInitData{ImageURL: imageURL, TargetDevice: targetDevice}
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return buf.String(), nil
}

func (h *Hetzner) fetchSnapshotByName(ctx context.Context, name string) (*hcloud.Image, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("image name is required")
	}

	if id, err := strconv.ParseInt(name, 10, 64); err == nil {
		image, _, err := h.Client.Image.GetByID(ctx, id)
		return image, err
	}

	selector := fmt.Sprintf("%s,%s=%s", managedLabelSelector(), opsImageNameLabelKey, name)
	opts := hcloud.ImageListOpts{
		ListOpts: hcloud.ListOpts{LabelSelector: selector},
		Type:     []hcloud.ImageType{hcloud.ImageTypeSnapshot},
	}

	images, err := h.Client.Image.AllWithOpts(ctx, opts)
	if err != nil {
		return nil, err
	}

	for _, img := range images {
		if labelValue(img.Labels, opsImageNameLabelKey) == name || img.Name == name {
			return img, nil
		}
	}

	return nil, nil
}

func (h *Hetzner) resolveImageReference(ctx context.Context, ref string) (*hcloud.Image, error) {
	if ref == "" {
		return nil, fmt.Errorf("image reference is empty")
	}

	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		image, _, err := h.Client.Image.GetByID(ctx, id)
		return image, err
	}

	image, _, err := h.Client.Image.GetByName(ctx, ref)
	if err != nil {
		return nil, err
	}
	return image, nil
}

func (h *Hetzner) createEphemeralSSHKey(ctx context.Context) (*hcloud.SSHKey, error) {
	_, publicKey, err := GenerateSSHKeyPair()
	if err != nil {
		return nil, err
	}

	name := fmt.Sprintf("ops-hetzner-%d", time.Now().UnixNano())
	sshKey, _, err := h.Client.SSHKey.Create(ctx, hcloud.SSHKeyCreateOpts{
		Name:      name,
		PublicKey: publicKey,
	})
	return sshKey, err
}

func waitForServerStatus(ctx context.Context, client *hcloud.Client, serverID int64, desiredStatus hcloud.ServerStatus) error {
	pollTicker := time.NewTicker(10 * time.Second)
	defer pollTicker.Stop()

	timeout := time.NewTimer(5 * time.Minute)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return fmt.Errorf("timed out waiting for server %d to reach status %s", serverID, desiredStatus)
		case <-pollTicker.C:
			server, _, err := client.Server.GetByID(ctx, serverID)
			if err != nil {
				return err
			}
			if server == nil {
				return fmt.Errorf("server %d not found", serverID)
			}
			if server.Status == desiredStatus {
				return nil
			}
		}
	}
}

func combineLabels(base map[string]string, tags []types.Tag, predicate func(types.Tag) bool) map[string]string {
	result := make(map[string]string, len(base)+len(tags))
	for k, v := range base {
		result[k] = v
	}

	for _, tag := range tags {
		if !predicate(tag) {
			continue
		}
		if tag.Key == "" || tag.Value == "" {
			continue
		}
		key := sanitizeLabelKey(tag.Key)
		if key == "" {
			continue
		}
		result[key] = sanitizeLabelValue(tag.Value)
	}

	return result
}

func flattenLabels(labels map[string]string) []string {
	if len(labels) == 0 {
		return nil
	}
	result := make([]string, 0, len(labels))
	for k, v := range labels {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}

func labelValue(labels map[string]string, key string) string {
	if labels == nil {
		return ""
	}
	if val, ok := labels[key]; ok {
		return val
	}
	return ""
}

func sanitizeLabelKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	key = strings.ReplaceAll(key, " ", "-")
	key = strings.ReplaceAll(key, "/", "-")
	key = strings.ReplaceAll(key, ":", "-")
	return key
}

func sanitizeLabelValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, " ", "-")
	return value
}

func jsonOutput(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// GenerateSSHKeyPair generates a new RSA key pair and returns the PEM-encoded private key
// and the OpenSSH-formatted public key as strings.
func GenerateSSHKeyPair() (string, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate RSA private key: %w", err)
	}

	privateKeyDER := x509.MarshalPKCS1PrivateKey(privateKey)
	privatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyDER,
	})
	if privatePEM == nil {
		return "", "", fmt.Errorf("failed to encode private key to PEM format")
	}

	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to create SSH public key: %w", err)
	}

	publicPEM := ssh.MarshalAuthorizedKey(publicKey)

	return string(privatePEM), string(publicPEM), nil
}
