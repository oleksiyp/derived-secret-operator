/*
Copyright 2025.

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

package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	secretsv1alpha1 "github.com/oleksiyp/derived-secret-operator/api/v1alpha1"
	"github.com/oleksiyp/derived-secret-operator/internal/crypto"
)

const (
	masterPasswordKey = "masterPassword"
	defaultLength     = 86
)

// MasterPasswordReconciler reconciles a MasterPassword object
type MasterPasswordReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	OperatorNamespace string
}

// +kubebuilder:rbac:groups=secrets.oleksiyp.dev,resources=masterpasswords,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=secrets.oleksiyp.dev,resources=masterpasswords/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=secrets.oleksiyp.dev,resources=derivedsecrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *MasterPasswordReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the MasterPassword instance
	masterPassword := &secretsv1alpha1.MasterPassword{}
	err := r.Get(ctx, req.NamespacedName, masterPassword)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("MasterPassword resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get MasterPassword")
		return ctrl.Result{}, err
	}

	// Reconcile the secret
	if err := r.reconcileSecret(ctx, masterPassword); err != nil {
		log.Error(err, "Failed to reconcile secret")
		r.setCondition(masterPassword, "Ready", metav1.ConditionFalse, "SecretReconciliationFailed", err.Error())
		if err := r.Status().Update(ctx, masterPassword); err != nil {
			log.Error(err, "Failed to update status")
		}
		return ctrl.Result{}, err
	}

	// Update status
	if err := r.updateStatus(ctx, masterPassword); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled MasterPassword")
	return ctrl.Result{}, nil
}

// reconcileSecret ensures the master password secret exists and is up to date
func (r *MasterPasswordReconciler) reconcileSecret(ctx context.Context, mp *secretsv1alpha1.MasterPassword) error {
	log := logf.FromContext(ctx)

	secretName, secretNamespace := r.getSecretNameAndNamespace(mp)

	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: secretNamespace}, secret)

	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get secret: %w", err)
		}

		// Secret doesn't exist, check if we should create it
		if mp.Spec.Secret != nil && !mp.Spec.Secret.Create {
			return fmt.Errorf("secret %s/%s does not exist and create is false", secretNamespace, secretName)
		}

		// Generate a new master password
		length := mp.Spec.Length
		if length == 0 {
			length = defaultLength
		}

		password, err := crypto.GenerateRandomPassword(length)
		if err != nil {
			return fmt.Errorf("failed to generate master password: %w", err)
		}

		// Create the secret
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        secretName,
				Namespace:   secretNamespace,
				Labels:      map[string]string{"app.kubernetes.io/managed-by": "derived-secret-operator"},
				Annotations: mp.Spec.Annotations,
			},
			Type: corev1.SecretTypeOpaque,
			StringData: map[string]string{
				masterPasswordKey: password,
			},
		}

		if err := r.Create(ctx, secret); err != nil {
			return fmt.Errorf("failed to create secret: %w", err)
		}

		log.Info("Created master password secret", "secret", secretNamespace+"/"+secretName)
		return nil
	}

	// Secret exists, ensure it has the master password key
	if _, ok := secret.Data[masterPasswordKey]; !ok {
		return fmt.Errorf("secret %s/%s exists but missing %s key", secretNamespace, secretName, masterPasswordKey)
	}

	// Update annotations if they changed
	if mp.Spec.Annotations != nil {
		needsUpdate := false
		if secret.Annotations == nil {
			secret.Annotations = make(map[string]string)
		}
		for k, v := range mp.Spec.Annotations {
			if secret.Annotations[k] != v {
				secret.Annotations[k] = v
				needsUpdate = true
			}
		}
		if needsUpdate {
			if err := r.Update(ctx, secret); err != nil {
				return fmt.Errorf("failed to update secret annotations: %w", err)
			}
			log.Info("Updated secret annotations", "secret", secretNamespace+"/"+secretName)
		}
	}

	return nil
}

// updateStatus updates the MasterPassword status
func (r *MasterPasswordReconciler) updateStatus(ctx context.Context, mp *secretsv1alpha1.MasterPassword) error {
	log := logf.FromContext(ctx)

	secretName, secretNamespace := r.getSecretNameAndNamespace(mp)

	// Count dependent DerivedSecrets
	derivedSecrets := &secretsv1alpha1.DerivedSecretList{}
	if err := r.List(ctx, derivedSecrets); err != nil {
		return fmt.Errorf("failed to list DerivedSecrets: %w", err)
	}

	dependentCount := 0
	for _, ds := range derivedSecrets.Items {
		for _, keySpec := range ds.Spec.Keys {
			mpName := keySpec.MasterPassword
			if mpName == "" {
				mpName = "default"
			}
			if mpName == mp.Name {
				dependentCount++
				break
			}
		}
	}

	mp.Status.SecretName = secretName
	mp.Status.SecretNamespace = secretNamespace
	mp.Status.Ready = true
	mp.Status.DependentSecrets = dependentCount

	r.setCondition(mp, "Ready", metav1.ConditionTrue, "SecretReady", "Master password secret is ready")

	if err := r.Status().Update(ctx, mp); err != nil {
		log.Error(err, "Failed to update status")
		return err
	}

	return nil
}

// getSecretNameAndNamespace returns the secret name and namespace for the MasterPassword
func (r *MasterPasswordReconciler) getSecretNameAndNamespace(mp *secretsv1alpha1.MasterPassword) (string, string) {
	secretName := mp.Name + "-mp"
	secretNamespace := r.OperatorNamespace

	if mp.Spec.Secret != nil && mp.Spec.Secret.Name != "" {
		secretName = mp.Spec.Secret.Name
	}

	return secretName, secretNamespace
}

// setCondition sets a condition on the MasterPassword
func (r *MasterPasswordReconciler) setCondition(
	mp *secretsv1alpha1.MasterPassword,
	condType string,
	status metav1.ConditionStatus,
	reason, message string,
) {
	condition := metav1.Condition{
		Type:               condType,
		Status:             status,
		ObservedGeneration: mp.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
	meta.SetStatusCondition(&mp.Status.Conditions, condition)
}

// findMasterPasswordsForSecret returns an event handler that maps Secret events to MasterPassword reconcile requests
func (r *MasterPasswordReconciler) findMasterPasswordsForSecret() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []ctrl.Request {
		secret, ok := obj.(*corev1.Secret)
		if !ok {
			return nil
		}

		// Only watch secrets in the operator namespace
		if secret.Namespace != r.OperatorNamespace {
			return nil
		}

		// If the secret has our label, it's definitely managed by us
		// If not, still check if it matches any MasterPassword (for deletion events where labels may be gone)
		isManagedByUs := secret.Labels != nil && secret.Labels["app.kubernetes.io/managed-by"] == "derived-secret-operator"

		// List all MasterPasswords to find which one corresponds to this secret
		mpList := &secretsv1alpha1.MasterPasswordList{}
		if err := r.List(ctx, mpList); err != nil {
			return nil
		}

		var requests []ctrl.Request
		for _, mp := range mpList.Items {
			secretName, secretNamespace := r.getSecretNameAndNamespace(&mp)
			if secret.Name == secretName && secret.Namespace == secretNamespace {
				// Only trigger reconcile if:
				// 1. Secret has our label (create/update/delete of managed secret)
				// 2. OR secret name matches the expected pattern (catch deletion events)
				if isManagedByUs || secret.Name == mp.Name+"-mp" {
					requests = append(requests, ctrl.Request{
						NamespacedName: types.NamespacedName{
							Name: mp.Name,
						},
					})
				}
			}
		}

		return requests
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *MasterPasswordReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&secretsv1alpha1.MasterPassword{}).
		Watches(&corev1.Secret{}, r.findMasterPasswordsForSecret()).
		Named("masterpassword").
		Complete(r)
}
