package host

import (
	"context"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/deckhouse/virtualization-csi-driver/internal/entities"
	"github.com/deckhouse/virtualization/api/core/v1alpha2"
)

func (c *Client) CreateDisk(ctx context.Context, name string, size int64, storageClass *string) (*entities.Disk, error) {
	vmd := v1alpha2.VirtualMachineDisk{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha2.VMDKind,
			APIVersion: v1alpha2.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: c.namespace,
		},
		Spec: v1alpha2.VirtualMachineDiskSpec{
			PersistentVolumeClaim: v1alpha2.VMDPersistentVolumeClaim{
				StorageClassName: storageClass,
				Size:             resource.NewQuantity(size, resource.BinarySI),
			},
		},
	}

	err := c.crClient.Create(ctx, &vmd)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return nil, err
	}

	return &entities.Disk{
		Name: vmd.Name,
	}, nil
}

func (c *Client) WaitDiskCreation(ctx context.Context, vmdName string) error {
	return c.Wait(ctx, vmdName, &v1alpha2.VirtualMachineDisk{}, func(obj client.Object) (bool, error) {
		vmd, ok := obj.(*v1alpha2.VirtualMachineDisk)
		if !ok {
			return false, fmt.Errorf("expected a VirtualMachineDisk but got a %T", obj)
		}

		return vmd.Status.Phase == v1alpha2.DiskReady, nil
	})
}
