package host

import (
	"context"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

	"github.com/deckhouse/virtualization/api/core/v1alpha2"
)

func (c *Client) UpdateDiskCapacity(ctx context.Context, vmdName string, capacity *resource.Quantity) error {
	var vmd v1alpha2.VirtualMachineDisk

	err := c.crClient.Get(ctx, types.NamespacedName{
		Namespace: c.namespace,
		Name:      vmdName,
	}, &vmd)
	if err != nil {
		return err
	}

	vmd.Spec.PersistentVolumeClaim.Size = capacity

	err = c.crClient.Update(ctx, &vmd)
	if err != nil {
		return err
	}

	return nil
}
