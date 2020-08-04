/*
Copyright 2020 The Kubernetes Authors.

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

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/mcs-api/pkg/apis/v1alpha1"
)

// ServiceImportReconciler reconciles a ServiceImport object
type ServiceImportReconciler struct {
	client.Client
	Log logr.Logger
}

// +kubebuilder:rbac:groups=multicluster.x-k8s.io,resources=serviceimports,verbs=get;list;watch;update;patch

func servicePorts(svcImport *v1alpha1.ServiceImport) []v1.ServicePort {
	ports := make([]v1.ServicePort, len(svcImport.Spec.Ports))
	for i, p := range svcImport.Spec.Ports {
		ports[i] = v1.ServicePort{
			Name:        p.Name,
			Protocol:    p.Protocol,
			Port:        p.Port,
			AppProtocol: p.AppProtocol,
		}
	}
	return ports
}

func shouldIgnoreImport(svcImport *v1alpha1.ServiceImport) bool {
	if svcImport.DeletionTimestamp != nil {
		return true
	}
	if svcImport.Spec.Type != v1alpha1.ClusterSetIP {
		return true
	}
	return false
}

// Reconcile the changes.
func (r *ServiceImportReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serviceName := derivedName(req.NamespacedName)
	log := r.Log.WithValues("serviceimport", req.NamespacedName, "derived", serviceName)
	var svcImport v1alpha1.ServiceImport
	if err := r.Client.Get(ctx, req.NamespacedName, &svcImport); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if shouldIgnoreImport(&svcImport) {
		return ctrl.Result{}, nil
	}

	// Ensure the existence of the derived service
	var svc v1.Service
	if svcImport.Annotations[DerivedServiceAnnotation] == "" {
		if svcImport.Annotations == nil {
			svcImport.Annotations = map[string]string{}
		}
		svcImport.Annotations[DerivedServiceAnnotation] = derivedName(req.NamespacedName)
		if err := r.Client.Update(ctx, &svcImport); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("added annotation", DerivedServiceAnnotation, svcImport.Annotations[DerivedServiceAnnotation])
		return ctrl.Result{}, nil
	}
	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: svcImport.Annotations[DerivedServiceAnnotation]}, &svc); err == nil {
		return ctrl.Result{}, nil
	} else if !apierrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}
	svc = v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.Namespace,
			Name:      svcImport.Annotations[DerivedServiceAnnotation],
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       req.Name,
					Kind:       serviceImportKind,
					APIVersion: v1alpha1.GroupVersion.String(),
					UID:        svcImport.UID,
				},
			},
		},
		Spec: v1.ServiceSpec{
			Type:  v1.ServiceTypeClusterIP,
			Ports: servicePorts(&svcImport),
		},
	}
	if err := r.Client.Create(ctx, &svc); err != nil {
		return ctrl.Result{}, err
	}
	log.Info("created service")
	return ctrl.Result{}, nil
}

// SetupWithManager wires up the controller.
func (r *ServiceImportReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.ServiceImport{}).Complete(r)
}
