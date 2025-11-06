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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	secretsv1alpha1 "github.com/oleksiyp/derived-secret-operator/api/v1alpha1"
	"github.com/oleksiyp/derived-secret-operator/internal/crypto"
)

const (
	derivedSecretFinalizer = "secrets.oleksiyp.dev/derivedsecret-finalizer"
)

// DerivedSecretReconciler reconciles a DerivedSecret object
type DerivedSecretReconciler struct {
	client.Client
	Scheme            *runtime.Scheme
	OperatorNamespace string
}

// +kubebuilder:rbac:groups=secrets.oleksiyp.dev,resources=derivedsecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=secrets.oleksiyp.dev,resources=derivedsecrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=secrets.oleksiyp.dev,resources=derivedsecrets/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=secrets.oleksiyp.dev,resources=masterpasswords,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *DerivedSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the DerivedSecret instance
	derivedSecret := &secretsv1alpha1.DerivedSecret{}
	err := r.Get(ctx, req.NamespacedName, derivedSecret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("DerivedSecret resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get DerivedSecret")
		return ctrl.Result{}, err
	}

	// Check if the DerivedSecret is being deleted
	if !derivedSecret.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, derivedSecret)
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(derivedSecret, derivedSecretFinalizer) {
		controllerutil.AddFinalizer(derivedSecret, derivedSecretFinalizer)
		if err := r.Update(ctx, derivedSecret); err != nil {
			log.Error(err, "Failed to add finalizer to DerivedSecret")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Reconcile the derived secret
	if err := r.reconcileDerivedSecret(ctx, derivedSecret); err != nil {
		log.Error(err, "Failed to reconcile derived secret")
		r.setCondition(derivedSecret, "Ready", metav1.ConditionFalse, "ReconciliationFailed", err.Error())
		if err := r.Status().Update(ctx, derivedSecret); err != nil {
			log.Error(err, "Failed to update status")
		}
		return ctrl.Result{}, err
	}

	// Update status
	if err := r.updateStatus(ctx, derivedSecret); err != nil {
		log.Error(err, "Failed to update status")
		return ctrl.Result{}, err
	}

	log.Info("Successfully reconciled DerivedSecret")
	return ctrl.Result{}, nil
}

// reconcileDerivedSecret reconciles the actual Kubernetes secret based on the DerivedSecret spec
func (r *DerivedSecretReconciler) reconcileDerivedSecret(ctx context.Context, ds *secretsv1alpha1.DerivedSecret) error {
	log := logf.FromContext(ctx)

	// Derive all secrets
	secretData := make(map[string][]byte)
	for keyName, keySpec := range ds.Spec.Keys {
		masterPasswordName := keySpec.MasterPassword
		if masterPasswordName == "" {
			masterPasswordName = "default"
		}

		// Get the master password
		masterPassword, err := r.getMasterPassword(ctx, masterPasswordName)
		if err != nil {
			return fmt.Errorf("failed to get master password %s: %w", masterPasswordName, err)
		}

		// Derive the secret
		length := crypto.GetSecretLength(string(keySpec.Type), keySpec.Length)
		context := crypto.BuildContext(ds.Namespace, ds.Name, keyName)

		derivedValue, err := crypto.DeriveSecret(masterPassword, context, length)
		if err != nil {
			return fmt.Errorf("failed to derive secret for key %s: %w", keyName, err)
		}

		secretData[keyName] = []byte(derivedValue)
	}

	// Create or update the Kubernetes secret
	secret := &corev1.Secret{}
	secretName := ds.Name
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: ds.Namespace}, secret)

	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get secret: %w", err)
		}

		// Create new secret
		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        secretName,
				Namespace:   ds.Namespace,
				Labels:      ds.Spec.Labels,
				Annotations: ds.Spec.Annotations,
			},
			Type: ds.Spec.Type,
			Data: secretData,
		}

		// Set owner reference
		if err := controllerutil.SetControllerReference(ds, secret, r.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference: %w", err)
		}

		if err := r.Create(ctx, secret); err != nil {
			return fmt.Errorf("failed to create secret: %w", err)
		}

		log.Info("Created derived secret", "secret", ds.Namespace+"/"+secretName)
		return nil
	}

	// Update existing secret
	needsUpdate := false

	// Check if data changed
	if !equalSecretData(secret.Data, secretData) {
		secret.Data = secretData
		needsUpdate = true
	}

	// Check if type changed
	if secret.Type != ds.Spec.Type {
		secret.Type = ds.Spec.Type
		needsUpdate = true
	}

	// Update labels
	if !equalMaps(secret.Labels, ds.Spec.Labels) {
		secret.Labels = ds.Spec.Labels
		needsUpdate = true
	}

	// Update annotations
	if !equalMaps(secret.Annotations, ds.Spec.Annotations) {
		secret.Annotations = ds.Spec.Annotations
		needsUpdate = true
	}

	if needsUpdate {
		if err := r.Update(ctx, secret); err != nil {
			return fmt.Errorf("failed to update secret: %w", err)
		}
		log.Info("Updated derived secret", "secret", ds.Namespace+"/"+secretName)
	}

	return nil
}

// getMasterPassword fetches the master password from the MasterPassword resource
func (r *DerivedSecretReconciler) getMasterPassword(ctx context.Context, name string) (string, error) {
	// Fetch the MasterPassword resource
	masterPassword := &secretsv1alpha1.MasterPassword{}
	if err := r.Get(ctx, types.NamespacedName{Name: name}, masterPassword); err != nil {
		return "", fmt.Errorf("failed to get MasterPassword %s: %w", name, err)
	}

	// Get the secret name and namespace
	secretName := masterPassword.Name + "-mp"
	if masterPassword.Spec.Secret != nil && masterPassword.Spec.Secret.Name != "" {
		secretName = masterPassword.Spec.Secret.Name
	}
	secretNamespace := r.OperatorNamespace

	// Fetch the secret
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: secretNamespace}, secret); err != nil {
		return "", fmt.Errorf("failed to get master password secret %s/%s: %w", secretNamespace, secretName, err)
	}

	// Extract the master password
	passwordBytes, ok := secret.Data[masterPasswordKey]
	if !ok {
		return "", fmt.Errorf("master password secret %s/%s missing key %s", secretNamespace, secretName, masterPasswordKey)
	}

	return string(passwordBytes), nil
}

// handleDeletion handles the deletion of a DerivedSecret
func (r *DerivedSecretReconciler) handleDeletion(ctx context.Context, ds *secretsv1alpha1.DerivedSecret) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(ds, derivedSecretFinalizer) {
		return ctrl.Result{}, nil
	}

	// Delete the associated secret
	secret := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: ds.Name, Namespace: ds.Namespace}, secret)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "Failed to get secret for deletion")
			return ctrl.Result{}, err
		}
		// Secret already deleted
	} else {
		// Delete the secret
		if err := r.Delete(ctx, secret); err != nil {
			log.Error(err, "Failed to delete secret")
			return ctrl.Result{}, err
		}
		log.Info("Deleted derived secret", "secret", ds.Namespace+"/"+ds.Name)
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(ds, derivedSecretFinalizer)
	if err := r.Update(ctx, ds); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	log.Info("Finalizer removed, DerivedSecret will be deleted")
	return ctrl.Result{}, nil
}

// updateStatus updates the DerivedSecret status
func (r *DerivedSecretReconciler) updateStatus(ctx context.Context, ds *secretsv1alpha1.DerivedSecret) error {
	log := logf.FromContext(ctx)

	ds.Status.SecretName = ds.Name
	ds.Status.Ready = true
	now := metav1.Now()
	ds.Status.LastUpdated = &now

	r.setCondition(ds, "Ready", metav1.ConditionTrue, "SecretReady", "Derived secret is ready")

	if err := r.Status().Update(ctx, ds); err != nil {
		log.Error(err, "Failed to update status")
		return err
	}

	return nil
}

// setCondition sets a condition on the DerivedSecret
func (r *DerivedSecretReconciler) setCondition(ds *secretsv1alpha1.DerivedSecret, condType string, status metav1.ConditionStatus, reason, message string) {
	condition := metav1.Condition{
		Type:               condType,
		Status:             status,
		ObservedGeneration: ds.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
	meta.SetStatusCondition(&ds.Status.Conditions, condition)
}

// equalSecretData compares two secret data maps
func equalSecretData(a, b map[string][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || string(v) != string(bv) {
			return false
		}
	}
	return true
}

// equalMaps compares two string maps
func equalMaps(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || v != bv {
			return false
		}
	}
	return true
}

// SetupWithManager sets up the controller with the Manager.
func (r *DerivedSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&secretsv1alpha1.DerivedSecret{}).
		Owns(&corev1.Secret{}).
		Named("derivedsecret").
		Complete(r)
}
