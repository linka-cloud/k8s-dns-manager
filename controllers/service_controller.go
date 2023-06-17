/*
Copyright 2020 The Linka Cloud Team.

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
	"strconv"

	"github.com/go-logr/logr"
	"github.com/weppos/publicsuffix-go/publicsuffix"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	dnsv1alpha1 "go.linka.cloud/k8s/dns/api/v1alpha1"
)

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("service", req.NamespacedName)
	var svc corev1.Service
	if err := r.Get(ctx, req.NamespacedName, &svc); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		// garbage collection should delete the DNSRecord
		return ctrl.Result{}, nil
	}

	var recs dnsv1alpha1.DNSRecordList
	if err := r.List(ctx, &recs, client.InNamespace(req.Namespace)); err != nil {
		log.Error(err, "unable to list DNSRecords")
		return ctrl.Result{}, err
	}
	got, err := childRecords(ctx, r.Client, &svc, ServiceAnnotation)
	if err != nil {
		log.Error(err, "unable to get child DNSRecords")
		return ctrl.Result{}, err
	}
	if svc.Annotations == nil {
		svc.Annotations = make(map[string]string)
	}
	var hostname string
	if v, ok := svc.Annotations[HostnameAnnotation]; ok {
		hostname = v
	}
	if _, err := publicsuffix.Domain(hostname); hostname != "" && err != nil {
		log.Error(err, "invalid hostname", "hostname", hostname)
		hostname = ""
	}
	if _, ok := svc.Annotations[IgnoredAnnotation]; ok || svc.Spec.Type != corev1.ServiceTypeLoadBalancer || hostname == "" {
		for _, v := range got.Items {
			if err := r.Delete(ctx, &v); err != nil {
				if client.IgnoreNotFound(err) != nil {
					log.Error(err, "unable to delete DNSRecord", "name", v.Name)
					return ctrl.Result{}, err
				}
			}
		}
		return ctrl.Result{}, nil
	}

	var ttl uint32
	if v, ok := svc.Annotations[TTLAnnotation]; ok {
		i, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			log.Error(err, "invalid TTL annotation, using defaults: 3600")
		} else {
			ttl = uint32(i)
		}
	}
	var want dnsv1alpha1.DNSRecordList
	var ips []string
	for _, v := range svc.Status.LoadBalancer.Ingress {
		if v.IP != "" {
			ips = append(ips, v.IP)
		}
	}
	for i, vv := range ips {
		rec := dnsv1alpha1.DNSRecord{
			ObjectMeta: metav1.ObjectMeta{
				Name:      recordName(svc.Name, "svc", hostname, i),
				Namespace: svc.Namespace,
				Annotations: map[string]string{
					ServiceAnnotation: svc.Name,
				},
			},
			Spec: dnsv1alpha1.DNSRecordSpec{
				A: &dnsv1alpha1.ARecord{
					Name:   hostname,
					Ttl:    ttl,
					Target: vv,
				},
			},
		}
		rec.Default()
		if err := ctrl.SetControllerReference(&svc, &rec, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		want.Items = append(want.Items, rec)
	}

	return reconcileChildRecords(ctrl.LoggerInto(ctx, log), r.Client, got, want)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	fn := extractValue("core/v1", "Service")
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Service{}, ownerKey, fn); err != nil {
		return err
	}
	filter := func(o client.Object) bool {
		if o == nil {
			return false
		}
		svc, ok := o.(*corev1.Service)
		if !ok {
			return false
		}
		if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
			return false
		}
		if svc.Annotations == nil {
			return false
		}
		_, ok = svc.Annotations[HostnameAnnotation]
		return ok
	}
	p := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return filter(e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return filter(e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			if filter(e.ObjectOld) {
				return true
			}
			return filter(e.ObjectNew)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return filter(e.Object)
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		WithEventFilter(p).
		Owns(&dnsv1alpha1.DNSRecord{}).
		Complete(r)
}
