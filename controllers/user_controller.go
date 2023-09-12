/*
Copyright 2023 Francesco Ilario.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kimiov1alpha1 "github.com/filariow/kim/api/v1alpha1"
)

// UserReconciler reconciles a User object
type UserReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups="",namespace=system,resources=serviceaccounts,verbs=create;update;delete;get;list;watch
//+kubebuilder:rbac:groups="",namespace=system,resources=secrets,verbs=create;update;delete;get;list;watch
//+kubebuilder:rbac:groups=kim.io,namespace=system,resources=users,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kim.io,namespace=system,resources=users/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kim.io,namespace=system,resources=users/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the User object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx).WithValues("namespace", req.Namespace, "user", req.Name)

	// fetch user
	var u kimiov1alpha1.User
	if err := r.Get(ctx, req.NamespacedName, &u); err != nil {
		if errors.IsNotFound(err) {
			l.Info("user has been deleted")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// initialize the user
	nu := u.IsNewUser()
	if nu {
		u.Status.InitialGeneration = &u.ObjectMeta.Generation
	}

	switch u.Spec.State {
	case kimiov1alpha1.WaitingForApprovalUserState:
		// Nothing to do if user Is WaitingForApproval
		l.Info("user needs to be approved")
		if nu {
			return ctrl.Result{}, r.Status().Update(ctx, &u)
		}
		return ctrl.Result{}, nil

	case kimiov1alpha1.ActiveUserState:
		l.Info("user is active, ensure ServiceAccount and Secret exist")
		// Create the ServiceAccount and Secret if they don't exist
		if err := r.ensureServiceAccountAndSecretExist(ctx, &u); err != nil {
			l.Error(err, "error ensuring ServiceAccount and Secret exist")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, r.Status().Update(ctx, &u)

	case kimiov1alpha1.SuspendedUserState:
		// Delete the ServiceAccount
		l.Info("user is suspended, ensure ServiceAccount and Secret don't exist")
		if err := r.ensureServiceAccountDoesntExist(ctx, &u); err != nil {
			l.Error(err, "error ensuring ServiceAccount and Secret doen't exist")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, r.Status().Update(ctx, &u)

	case kimiov1alpha1.BannedUserState:
		// Delete the ServiceAccount
		l.Info("user is banned, ensure ServiceAccount and Secret don't exist")
		if err := r.ensureServiceAccountDoesntExist(ctx, &u); err != nil {
			l.Error(err, "error ensuring ServiceAccount and Secret doen't exist")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, r.Status().Update(ctx, &u)

	default:
		// That should not happen as for CRDs validation
		return ctrl.Result{}, nil
	}
}

func (r *UserReconciler) ensureServiceAccountDoesntExist(ctx context.Context, user *kimiov1alpha1.User) error {
	sa := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: user.Namespace,
			Name:      user.Name,
		},
	}

	if err := r.Delete(ctx, &sa); err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func (r *UserReconciler) ensureServiceAccountAndSecretExist(ctx context.Context, user *kimiov1alpha1.User) error {
	sa := corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: user.Namespace,
			Name:      user.Name,
		},
	}
	if err := r.Create(ctx, &sa); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	// reload the ServiceAccount for kubernetes assigned fields (UID, etc..)
	sat := types.NamespacedName{Namespace: sa.Namespace, Name: sa.Name}
	if err := r.Get(ctx, sat, &sa); err != nil {
		return err
	}

	s := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      user.Name,
			Namespace: user.Namespace,
			Annotations: map[string]string{
				corev1.ServiceAccountNameKey: sa.Name,
			},
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, &s, func() error {
		s.ObjectMeta.Annotations[corev1.ServiceAccountNameKey] = sa.Name
		s.OwnerReferences = []metav1.OwnerReference{
			{
				APIVersion: "v1",
				Kind:       "ServiceAccount",
				Name:       sa.Name,
				UID:        sa.UID,
			},
		}
		return nil
	})

	return err
}

// SetupWithManager sets up the controller with the Manager.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kimiov1alpha1.User{}).
		Complete(r)
}
