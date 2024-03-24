package host

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/deckhouse/virtualization/api/core/v1alpha2"
)

const (
	attachmentDiskNameLabel    = "virtualMachineDiskName"
	attachmentMachineNameLabel = "virtualMachineName"
)

type Attachment struct {
	Name string
}

func (c *Client) AttachDisk(ctx context.Context, vmdName, vmName string) (*Attachment, error) {
	vmbda, err := c.getVMBDA(ctx, vmdName, vmName)
	if vmbda != nil && err == nil {
		return &Attachment{Name: vmbda.Name}, nil
	}

	if err != nil && !errors.Is(err, ErrAttachmentNotFound) {
		return nil, err
	}

	vmbda = &v1alpha2.VirtualMachineBlockDeviceAttachment{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha2.VMBDAKind,
			APIVersion: v1alpha2.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vmbda-" + uuid.New().String(),
			Namespace: c.namespace,
			Labels: map[string]string{
				attachmentDiskNameLabel:    vmdName,
				attachmentMachineNameLabel: vmName,
			},
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

	return &Attachment{Name: vmbda.Name}, nil
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
	selector, err := labels.Parse(fmt.Sprintf("%s=%s,%s=%s", attachmentDiskNameLabel, vmdName, attachmentMachineNameLabel, vmName))
	if err != nil {
		return nil, err
	}

	var vmbdas v1alpha2.VirtualMachineBlockDeviceAttachmentList
	err = c.crClient.List(ctx, &vmbdas, &client.ListOptions{
		LabelSelector: selector,
		Namespace:     c.namespace,
	})
	if err != nil {
		return nil, err
	}

	if len(vmbdas.Items) == 0 {
		return nil, ErrAttachmentNotFound
	}

	if len(vmbdas.Items) != 1 {
		return nil, errors.New("more attachments found than expected: please report a bug")
	}

	return &vmbdas.Items[0], nil
}
