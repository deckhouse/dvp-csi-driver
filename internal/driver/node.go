package driver

import (
	"context"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ csi.NodeServer = &Driver{}

func (d *Driver) NodeStageVolume(_ context.Context, _ *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return &csi.NodeStageVolumeResponse{}, nil
}

func (d *Driver) NodeUnstageVolume(_ context.Context, _ *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return &csi.NodeUnstageVolumeResponse{}, nil
}

func (d *Driver) NodePublishVolume(_ context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	var mountOptions []string
	if req.GetReadonly() {
		mountOptions = append(mountOptions, "ro")
	}

	mnt := req.GetVolumeCapability().GetMount()
	if mnt != nil {
		mountOptions = append(mountOptions, mnt.GetMountFlags()...)
	}

	blockDevicePath, err := d.mounter.GetBlockDevicePathByID(req.VolumeId)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	switch req.GetVolumeCapability().GetAccessType().(type) {
	case *csi.VolumeCapability_Block:
		d.logger.Info("Mounting the volume block", "source", blockDevicePath, "target", req.GetTargetPath(), "opts", mountOptions)
		err = d.mounter.MountBlockDevice(blockDevicePath, req.GetTargetPath(), mountOptions...)
	case *csi.VolumeCapability_Mount:
		d.logger.Info("Mounting the volume file system", "source", blockDevicePath, "target", req.GetTargetPath(), "fs-type", mnt.GetFsType(), "opts", mountOptions)
		err = d.mounter.MountFileSystem(blockDevicePath, req.GetTargetPath(), mnt.GetFsType(), mountOptions...)
	default:
		return nil, status.Error(codes.InvalidArgument, "Unknown access type")
	}
	if err != nil {
		return nil, err
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

func (d *Driver) NodeUnpublishVolume(_ context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	err := d.mounter.Unmount(req.GetTargetPath())
	if err != nil {
		return nil, err
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

func (d *Driver) NodeGetVolumeStats(_ context.Context, _ *csi.NodeGetVolumeStatsRequest) (*csi.NodeGetVolumeStatsResponse, error) {
	return &csi.NodeGetVolumeStatsResponse{}, nil
}

func (d *Driver) NodeExpandVolume(_ context.Context, req *csi.NodeExpandVolumeRequest) (*csi.NodeExpandVolumeResponse, error) {
	volumeID := req.GetVolumeId()
	volumePath := req.GetVolumePath()
	if len(volumeID) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume id cannot be empty")
	}
	if len(volumePath) == 0 {
		return nil, status.Error(codes.InvalidArgument, "volume Path cannot be empty")
	}

	err := d.mounter.ResizeFS(volumePath)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &csi.NodeExpandVolumeResponse{
		// TODO: CapacityBytes: X,
	}, nil
}

func (d *Driver) NodeGetCapabilities(_ context.Context, _ *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	capabilities := []csi.NodeServiceCapability_RPC_Type{
		csi.NodeServiceCapability_RPC_EXPAND_VOLUME,
	}

	csiCaps := make([]*csi.NodeServiceCapability, len(capabilities))
	for i, capability := range capabilities {
		csiCaps[i] = &csi.NodeServiceCapability{
			Type: &csi.NodeServiceCapability_Rpc{
				Rpc: &csi.NodeServiceCapability_RPC{
					Type: capability,
				},
			},
		}
	}

	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: csiCaps,
	}, nil
}

const maxVolumesPerNodeTotal = 16

func (d *Driver) NodeGetInfo(ctx context.Context, _ *csi.NodeGetInfoRequest) (*csi.NodeGetInfoResponse, error) {
	count, err := d.hostCluster.GetBlockDeviceCount(ctx, d.nodeName)
	if err != nil {
		return nil, err
	}

	var maxVolumesPerNode int
	if count > 0 && count <= maxVolumesPerNodeTotal {
		maxVolumesPerNode = maxVolumesPerNodeTotal - count
	}

	return &csi.NodeGetInfoResponse{
		NodeId:             d.nodeName,
		MaxVolumesPerNode:  int64(maxVolumesPerNode),
		AccessibleTopology: &csi.Topology{},
	}, nil
}
