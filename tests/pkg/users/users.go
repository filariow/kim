package users

import (
	"context"
	"fmt"
	"time"

	"github.com/filariow/kim/tests/pkg/kube"
	"github.com/filariow/kim/tests/pkg/poll"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type Users struct {
	*kube.Kubernetes
}

func (u *Users) UserStateIs(ctx context.Context, name, state string) error {
	gvk := schema.GroupVersionKind{
		Group:   "kim.io",
		Version: "v1alpha1",
		Kind:    "User",
	}
	cli, err := u.Kubernetes.BuildNamespacedClientForResource(ctx, gvk, "")
	if err != nil {
		return err
	}

	lctx, cf := context.WithTimeout(ctx, 2*time.Minute)
	defer cf()

	return poll.Do(lctx, time.Second, func(ictx context.Context) error {
		r, err := cli.Get(ictx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		s, ok := r.Object["status"]
		if !ok {
			return fmt.Errorf("state not found for user %s", name)
		}

		ss, ok := s.(map[string]interface{})
		if !ok {
			return fmt.Errorf("user %s does not have a valid status: %v", name, s)
		}

		st, ok := ss["state"]
		if !ok {
			return fmt.Errorf("state not found in status of user %s: %v", name, s)
		}

		if st != state {
			return fmt.Errorf("user %s has state %s, wanted %s", name, st, state)
		}
		return nil
	})
}
