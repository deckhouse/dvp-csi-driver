package host

import (
	"context"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

	"github.com/deckhouse/virtualization/api/core/v1alpha2"
)

func (c *Client) GetDisk(ctx context.Context, vmdName string) (*Disk, error) {
	var vmd v1alpha2.VirtualMachineDisk

	err := c.crClient.Get(ctx, types.NamespacedName{
		Namespace: c.namespace,
		Name:      vmdName,
	}, &vmd)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, ErrDiskNotFound
		}

		return nil, err
	}

	err = c.crClient.Delete(ctx, &vmd)
	if err != nil {
		return nil, err
	}

	capacity, err := resource.ParseQuantity(vmd.Status.Capacity)
	if err != nil {
		return nil, err
	}

	return &Disk{
		Name:     vmd.Name,
		Capacity: capacity,
	}, nil
}
