package host

import (
	"context"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const defaultWaitCheckInterval = time.Second

type WaitFn func(obj client.Object) (bool, error)

func (c *Client) Wait(ctx context.Context, name string, obj client.Object, waitFn WaitFn) error {
	var done bool
	for {
		err := c.crClient.Get(ctx, types.NamespacedName{
			Namespace: c.namespace,
			Name:      name,
		}, obj)
		if err != nil {
			if !k8serrors.IsNotFound(err) {
				return err
			}

			// obj not found.
			done, err = waitFn(nil)
		} else {
			// obj found.
			done, err = waitFn(obj)
		}

		if err != nil {
			return err
		}

		if done {
			return nil
		}

		timer := time.NewTimer(defaultWaitCheckInterval)

		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		}
	}
}
