package host

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/deckhouse/virtualization/api/core/v1alpha2"

	"github.com/deckhouse/virtualization-csi-driver/internal/entities"
)

func (c *Client) AttachDisk(ctx context.Context, vmdName, vmName string) (*entities.Attachment, error) {
	vmbda, err := c.getVMBDA(ctx, vmdName, vmName)
	if err != nil && !errors.Is(err, ErrAttachmentNotFound) {
		return nil, err
	}

	if vmbda != nil && err == nil {
		return &entities.Attachment{Name: vmbda.Name}, nil
	}

	vmbda = &v1alpha2.VirtualMachineBlockDeviceAttachment{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha2.VMBDAKind,
			APIVersion: v1alpha2.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vmbda-" + uuid.New().String(),
			Namespace: c.namespace,
		},
		Spec: v1alpha2.VirtualMachineBlockDeviceAttachmentSpec{
			VMName: vmName,
			BlockDevice: v1alpha2.BlockDeviceAttachmentBlockDevice{
				Type: v1alpha2.BlockDeviceAttachmentTypeVirtualMachineDisk,
				VirtualMachineDisk: &v1alpha2.BlockDeviceAttachmentVirtualMachineDisk{
					Name: vmdName,
				},
			},
		},
	}

	err = c.crClient.Create(ctx, vmbda)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return nil, err
	}

	return &entities.Attachment{Name: vmbda.Name}, nil
}

func (c *Client) WaitDiskAttaching(ctx context.Context, attachmentName string) error {
	return c.Wait(ctx, attachmentName, &v1alpha2.VirtualMachineBlockDeviceAttachment{}, func(obj client.Object) (bool, error) {
		vmd, ok := obj.(*v1alpha2.VirtualMachineBlockDeviceAttachment)
		if !ok {
			return false, fmt.Errorf("expected a VirtualMachineBlockDeviceAttachment but got a %T", obj)
		}

		return vmd.Status.Phase == v1alpha2.BlockDeviceAttachmentPhaseAttached, nil
	})
}

func (c *Client) getVMBDA(ctx context.Context, vmdName, vmName string) (*v1alpha2.VirtualMachineBlockDeviceAttachment, error) {
	var vm v1alpha2.VirtualMachine

	err := c.crClient.Get(ctx, types.NamespacedName{
		Namespace: c.namespace,
		Name:      vmName,
	}, &vm)
	if err != nil {
		return nil, err
	}

	var vmbdaName string

	for _, bda := range vm.Status.BlockDevicesAttached {
		if !bda.Hotpluggable || bda.VirtualMachineBlockDeviceAttachment == nil {
			continue
		}

		if bda.Type != v1alpha2.DiskDevice || bda.VirtualMachineDisk == nil || bda.VirtualMachineDisk.Name != vmdName {
			continue
		}

		vmbdaName = bda.VirtualMachineBlockDeviceAttachment.Name
		break
	}

	if vmbdaName == "" {
		return nil, fmt.Errorf("disk %s isn't hot plugged to virtual machine %s: %w", vmdName, vmName, ErrAttachmentNotFound)
	}

	var vmbda v1alpha2.VirtualMachineBlockDeviceAttachment

	err = c.crClient.Get(ctx, types.NamespacedName{
		Name:      vmbdaName,
		Namespace: c.namespace,
	}, &vmbda)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, ErrAttachmentNotFound
		}

		return nil, err
	}

	return &vmbda, nil
}
