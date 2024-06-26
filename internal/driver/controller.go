package driver

import (
	"context"
	"errors"
	"fmt"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/deckhouse/dvp-csi-driver/internal/host"
)

var _ csi.ControllerServer = &Driver{}

func (d *Driver) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	for _, capability := range req.GetVolumeCapabilities() {
		switch capability.GetAccessMode().GetMode() {
		case csi.VolumeCapability_AccessMode_MULTI_NODE_READER_ONLY,
			csi.VolumeCapability_AccessMode_MULTI_NODE_SINGLE_WRITER,
			csi.VolumeCapability_AccessMode_MULTI_NODE_MULTI_WRITER,
			csi.VolumeCapability_AccessMode_UNKNOWN:
			return nil, status.Error(codes.InvalidArgument, "not supported pvc access mode")
		}
	}

	var storageClass *string
	dvpStorageClass, ok := req.GetParameters()["dvpStorageClass"]
	if ok {
		storageClass = &dvpStorageClass
	}

	disk, err := d.hostCluster.CreateDisk(ctx, req.Name, req.CapacityRange.RequiredBytes, storageClass)
	if err != nil {
		return nil, fmt.Errorf("failed to create disk: %w", err)
	}

	d.logger.Debug("Wait disk creation", "name", disk.Name)

	err = d.hostCluster.WaitDiskCreation(ctx, disk.Name)
	if err != nil {
		return nil, err
	}

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			CapacityBytes:      req.CapacityRange.RequiredBytes,
			VolumeId:           req.Name,
			VolumeContext:      map[string]string{},
			ContentSource:      req.VolumeContentSource,
			AccessibleTopology: []*csi.Topology{},
		},
	}, nil
}

// DeleteVolume TODO: deleting in process of creation.
func (d *Driver) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	disk, err := d.hostCluster.DeleteDisk(ctx, req.VolumeId)
	if err != nil {
		if errors.Is(err, host.ErrDiskAlreadyDeleted) {
			return &csi.DeleteVolumeResponse{}, nil
		}

		return nil, fmt.Errorf("failed to delete disk: %w", err)
	}

	d.logger.Debug("Wait disk deletion", "name", disk.Name)

	err = d.hostCluster.WaitDiskDeletion(ctx, disk.Name)
	if err != nil {
		return nil, err
	}

	return &csi.DeleteVolumeResponse{}, nil
}

func (d *Driver) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	attachment, err := d.hostCluster.AttachDisk(ctx, req.VolumeId, req.NodeId)
	if err != nil {
		return nil, fmt.Errorf("failed to create attachment: %w", err)
	}

	d.logger.Debug("Wait disk attaching", "name", attachment.Name)

	err = d.hostCluster.WaitDiskAttaching(ctx, attachment.Name)
	if err != nil {
		return nil, err
	}

	return &csi.ControllerPublishVolumeResponse{
		PublishContext: map[string]string{},
	}, nil
}

func (d *Driver) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	detachment, err := d.hostCluster.DetachDisk(ctx, req.VolumeId, req.NodeId)
	if err != nil {
		if errors.Is(err, host.ErrAttachmentAlreadyDeleted) {
			return &csi.ControllerUnpublishVolumeResponse{}, nil
		}

		return nil, fmt.Errorf("failed to delete attachment: %w", err)
	}

	d.logger.Debug("Wait disk detaching", "name", detachment.Name)

	err = d.hostCluster.WaitDiskDetaching(ctx, detachment.Name)
	if err != nil {
		return nil, err
	}

	return &csi.ControllerUnpublishVolumeResponse{}, nil
}

func (d *Driver) ValidateVolumeCapabilities(_ context.Context, _ *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	return nil, nil
}

func (d *Driver) ListVolumes(_ context.Context, _ *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	return nil, errors.New("not implemented")
}

func (d *Driver) GetCapacity(_ context.Context, _ *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return &csi.GetCapacityResponse{
		AvailableCapacity: 10 * 1024 * 1024,
	}, nil
}

func (d *Driver) ControllerGetCapabilities(_ context.Context, _ *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	capabilities := []csi.ControllerServiceCapability_RPC_Type{
		csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
		csi.ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
		csi.ControllerServiceCapability_RPC_EXPAND_VOLUME,
	}

	csiCaps := make([]*csi.ControllerServiceCapability, len(capabilities))
	for i, capability := range capabilities {
		csiCaps[i] = &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: capability,
				},
			},
		}
	}

	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: csiCaps,
	}, nil
}

func (d *Driver) CreateSnapshot(_ context.Context, _ *csi.CreateSnapshotRequest) (*csi.CreateSnapshotResponse, error) {
	return nil, errors.New("not implemented")
}

func (d *Driver) DeleteSnapshot(_ context.Context, _ *csi.DeleteSnapshotRequest) (*csi.DeleteSnapshotResponse, error) {
	return nil, errors.New("not implemented")
}

func (d *Driver) ListSnapshots(_ context.Context, _ *csi.ListSnapshotsRequest) (*csi.ListSnapshotsResponse, error) {
	return nil, errors.New("not implemented")
}

// ResizeDelta TODO: for what?
const ResizeDelta = "32Mi"

func (d *Driver) ControllerExpandVolume(ctx context.Context, req *csi.ControllerExpandVolumeRequest) (*csi.ControllerExpandVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "Volume id cannot be empty")
	}

	err := d.hostCluster.WaitDiskCreation(ctx, req.VolumeId)
	if err != nil {
		return nil, err
	}

	vmd, err := d.hostCluster.GetDisk(ctx, req.VolumeId)
	if err != nil {
		return nil, err
	}

	requiredCapacity := resource.NewQuantity(req.CapacityRange.GetRequiredBytes(), resource.BinarySI)

	nodeExpansionRequired := req.GetVolumeCapability().GetBlock() == nil

	if vmd.Capacity.Value() > requiredCapacity.Value() {
		// TODO: no decrease.
		return &csi.ControllerExpandVolumeResponse{
			CapacityBytes:         vmd.Capacity.Value(),
			NodeExpansionRequired: nodeExpansionRequired,
		}, nil
	}

	err = d.hostCluster.UpdateDiskCapacity(ctx, req.VolumeId, requiredCapacity)
	if err != nil {
		return nil, err
	}

	return &csi.ControllerExpandVolumeResponse{
		CapacityBytes:         req.CapacityRange.RequiredBytes,
		NodeExpansionRequired: nodeExpansionRequired,
	}, nil
}

func (d *Driver) ControllerGetVolume(_ context.Context, _ *csi.ControllerGetVolumeRequest) (*csi.ControllerGetVolumeResponse, error) {
	return nil, errors.New("not implemented")
}

func (d *Driver) ControllerModifyVolume(_ context.Context, _ *csi.ControllerModifyVolumeRequest) (*csi.ControllerModifyVolumeResponse, error) {
	return nil, errors.New("not implemented")
}
