package host

import (
	"context"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/deckhouse/virtualization/api/core/v1alpha2"
)

func (c *Client) GetBlockDeviceCount(ctx context.Context, vmName string) (int, error) {
	var vm v1alpha2.VirtualMachine

	err := c.crClient.Get(ctx, types.NamespacedName{
		Namespace: c.namespace,
		Name:      vmName,
	}, &vm)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return 0, ErrVMNotFound
		}

		return 0, err
	}

	return len(vm.Status.BlockDevicesAttached), nil
}
