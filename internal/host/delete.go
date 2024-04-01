package host

import (
	"context"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/deckhouse/virtualization-csi-driver/internal/entities"
	"github.com/deckhouse/virtualization/api/core/v1alpha2"
)

func (c *Client) DeleteDisk(ctx context.Context, vmdName string) (*entities.Disk, error) {
	var vmd v1alpha2.VirtualMachineDisk

	err := c.crClient.Get(ctx, types.NamespacedName{
		Namespace: c.namespace,
		Name:      vmdName,
	}, &vmd)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, ErrDiskAlreadyDeleted
		}

		return nil, err
	}

	err = c.crClient.Delete(ctx, &vmd)
	if err != nil {
		return nil, err
	}

	return &entities.Disk{Name: vmdName}, nil
}

func (c *Client) WaitDiskDeletion(ctx context.Context, vmdName string) error {
	return c.Wait(ctx, vmdName, &v1alpha2.VirtualMachineDisk{}, func(obj client.Object) (bool, error) {
		return obj == nil, nil
	})
}
