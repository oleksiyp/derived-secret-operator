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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	secretsv1alpha1 "github.com/oleksiyp/derived-secret-operator/api/v1alpha1"
)

var _ = Describe("DerivedSecret Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"
		const masterPasswordName = "default"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		derivedsecret := &secretsv1alpha1.DerivedSecret{}

		BeforeEach(func() {
			By("creating the MasterPassword and its secret")
			masterPasswordSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      masterPasswordName + "-mp",
					Namespace: "default", // Using default as OperatorNamespace for tests
				},
				Data: map[string][]byte{
					masterPasswordKey: []byte("test-master-password-for-testing-only"),
				},
			}
			secretNN := types.NamespacedName{
				Name:      masterPasswordSecret.Name,
				Namespace: masterPasswordSecret.Namespace,
			}
			err := k8sClient.Get(ctx, secretNN, &corev1.Secret{})
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, masterPasswordSecret)).To(Succeed())
			}

			masterPassword := &secretsv1alpha1.MasterPassword{
				ObjectMeta: metav1.ObjectMeta{
					Name: masterPasswordName,
				},
			}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: masterPasswordName}, &secretsv1alpha1.MasterPassword{})
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, masterPassword)).To(Succeed())
			}

			By("creating the custom resource for the Kind DerivedSecret")
			err = k8sClient.Get(ctx, typeNamespacedName, derivedsecret)
			if err != nil && errors.IsNotFound(err) {
				resource := &secretsv1alpha1.DerivedSecret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: secretsv1alpha1.DerivedSecretSpec{
						Keys: map[string]secretsv1alpha1.DerivedKeySpec{
							"password": {
								Type: secretsv1alpha1.SecretTypePassword,
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &secretsv1alpha1.DerivedSecret{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance DerivedSecret")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &DerivedSecretReconciler{
				Client:            k8sClient,
				Scheme:            k8sClient.Scheme(),
				OperatorNamespace: "default",
			}

			By("First reconcile adds finalizer")
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Second reconcile creates the secret")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking if the derived secret was created")
			secret := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: "default"}, secret)
			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Data).To(HaveKey("password"))
			Expect(secret.Data["password"]).To(HaveLen(26)) // password type is 26 chars
		})
	})
})
