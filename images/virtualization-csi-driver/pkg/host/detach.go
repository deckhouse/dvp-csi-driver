package host

import (
	"context"
	"errors"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/deckhouse/virtualization/api/core/v1alpha2"
)

func (c *Client) DetachDisk(ctx context.Context, vmdName, vmName string) (*Attachment, error) {
	vmbda, err := c.getVMBDA(ctx, vmdName, vmName)
	if err != nil {
		if errors.Is(err, ErrAttachmentNotFound) {
			return nil, ErrAttachmentAlreadyDeleted
		}

		return nil, err
	}

	err = c.crClient.Delete(ctx, vmbda)
	if err != nil {
		return nil, err
	}

	return &Attachment{Name: vmbda.Name}, nil
}

func (c *Client) WaitDiskDetaching(ctx context.Context, attachmentName string) error {
	return c.Wait(ctx, attachmentName, &v1alpha2.VirtualMachineBlockDeviceAttachment{}, func(obj client.Object) (bool, error) {
		return obj == nil, nil
	})
}
