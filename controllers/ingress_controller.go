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
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dnsv1alpha1 "go.linka.cloud/k8s/dns/api/v1alpha1"
)

const (
	HostnameAnnotation = "dns.linka.cloud/hostname"
	TTLAnnotation      = "dns.linka.cloud/ttl"
	IgnoredAnnotation  = "dns.linka.cloud/ignore"

	ownerKey = ".metadata.controller"
)

// IngressReconciler reconciles an Ingress object
type IngressReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=networking.k8s.io.linka.cloud,resources=ingresses,verbs=get;list;watch
// +kubebuilder:rbac:groups=networking.k8s.io.linka.cloud,resources=ingresses/status,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *IngressReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ingress", req.NamespacedName)
	var ing networkingv1.Ingress
	if err := r.Get(ctx, req.NamespacedName, &ing); err != nil {
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
	var got dnsv1alpha1.DNSRecordList
	for _, v := range recs.Items {
		if metav1.IsControlledBy(&v, &ing) {
			got.Items = append(got.Items, v)
		}
	}
	if _, ok := ing.Annotations[IgnoredAnnotation]; ok {
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
	if v, ok := ing.Annotations[TTLAnnotation]; ok {
		i, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			log.Error(err, "invalid TTL annotation, using defaults: 3600")
		} else {
			ttl = uint32(i)
		}
	}
	var want dnsv1alpha1.DNSRecordList
	var ips []string
	for _, v := range ing.Status.LoadBalancer.Ingress {
		if v.IP != "" {
			ips = append(ips, v.IP)
		}
	}
	for _, v := range ing.Spec.Rules {
		if v.Host == "" {
			continue
		}
		for _, vv := range ips {
			rec := dnsv1alpha1.DNSRecord{
				ObjectMeta: metav1.ObjectMeta{
					Name:      recordName(ing.Name, v.Host),
					Namespace: ing.Namespace,
				},
				Spec: dnsv1alpha1.DNSRecordSpec{
					A: &dnsv1alpha1.ARecord{
						Name:   v.Host,
						Ttl:    ttl,
						Target: vv,
					},
				},
			}
			rec.Default()
			if err := ctrl.SetControllerReference(&ing, &rec, r.Scheme); err != nil {
				return ctrl.Result{}, err
			}
			want.Items = append(want.Items, rec)
		}
	}
	add, update, del := diffRecords(got, want)
	for _, v := range del {
		if err := r.Delete(ctx, &v); err != nil {
			if client.IgnoreNotFound(err) != nil {
				log.Error(err, "unable to delete DNSRecord", "name", v.Name)
				return ctrl.Result{}, err
			}
		}
	}
	for _, v := range add {
		if err := r.Create(ctx, &v); err != nil {
			log.Error(err, "unable to create DNSRecord", "name", v.Name)
			return ctrl.Result{}, err
		}
	}
	for _, v := range update {
		if err := r.Update(ctx, &v); err != nil {
			log.Error(err, "unable to update DNSRecord", "name", v.Name)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &dnsv1alpha1.DNSRecord{}, ownerKey, extractValue); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		For(&networkingv1.Ingress{}).
		Owns(&dnsv1alpha1.DNSRecord{}).
		Complete(r)
}

func extractValue(rawObj client.Object) []string {
	// grab the owner object
	owner := metav1.GetControllerOf(rawObj)
	if owner == nil {
		return nil
	}

	if owner.APIVersion != "networking.k8s.io/v1beta1" || owner.Kind != "Ingress" {
		return nil
	}

	return []string{owner.Name}
}

func recordName(name, host string) string {
	return fmt.Sprintf("%s-%s", name, strings.Replace(host, ".", "-", -1))
}

func diffRecords(got, want dnsv1alpha1.DNSRecordList) (add, update, del []dnsv1alpha1.DNSRecord) {
	for _, v := range want.Items {
		found := false
		for _, vv := range got.Items {
			if v.Name == vv.Name {
				found = true
				if !reflect.DeepEqual(v.Spec, vv.Spec) {
					update = append(update, vv)
				}
				break
			}
		}
		if !found {
			add = append(add, v)
		}
	}
	for _, v := range got.Items {
		found := false
		for _, vv := range want.Items {
			if v.Name == vv.Name {
				found = true
				break
			}
		}
		if !found {
			del = append(del, v)
		}
	}
	return
}