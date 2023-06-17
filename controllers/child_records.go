package controllers

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dnsv1alpha1 "go.linka.cloud/k8s/dns/api/v1alpha1"
)

const (
	HostnameAnnotation = "dns.linka.cloud/hostname"
	TargetAnnotation   = "dns.linka.cloud/target"
	TTLAnnotation      = "dns.linka.cloud/ttl"
	IgnoredAnnotation  = "dns.linka.cloud/ignore"

	IngressAnnotation = "dns.linka.cloud/ingress"
	ServiceAnnotation = "dns.linka.cloud/service"

	ownerKey = ".metadata.controller"
)

func recordName(name, typ, host string, index int) string {
	return fmt.Sprintf("%s-%s-%s-%d", name, typ, strings.NewReplacer(".", "-", "*", "wildcard").Replace(host), index)
}

func childRecords(ctx context.Context, c client.Client, o client.Object, annotation string) (dnsv1alpha1.DNSRecordList, error) {
	log := ctrl.LoggerFrom(ctx)
	var recs dnsv1alpha1.DNSRecordList
	if err := c.List(ctx, &recs, client.InNamespace(o.GetNamespace())); err != nil {
		return dnsv1alpha1.DNSRecordList{}, err
	}
	var got dnsv1alpha1.DNSRecordList
	for _, v := range recs.Items {
		if metav1.IsControlledBy(&v, o) {
			got.Items = append(got.Items, v)
			continue
		}
		if len(v.Annotations) == 0 {
			continue
		}
		// we might have records without owner reference if resources were backed up and restored
		if n, ok := v.Annotations[annotation]; ok && n == o.GetName() {
			log.Info("found record without owner reference", "name", v.Name, "owner", o.GetName())
			if err := ctrl.SetControllerReference(o, &v, c.Scheme()); err != nil {
				return dnsv1alpha1.DNSRecordList{}, err
			}
			got.Items = append(got.Items, v)
			continue
		}
	}
	return got, nil
}

func reconcileChildRecords(ctx context.Context, c client.Client, got, want dnsv1alpha1.DNSRecordList) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	add, update, del := diffRecords(got, want)
	for _, v := range del {
		if err := c.Delete(ctx, &v); err != nil {
			if client.IgnoreNotFound(err) != nil {
				log.Error(err, "unable to delete DNSRecord", "name", v.Name)
				return ctrl.Result{}, err
			}
		}
	}
	for _, v := range add {
		if err := c.Create(ctx, &v); err != nil {
			log.Error(err, "unable to create DNSRecord", "name", v.Name)
			return ctrl.Result{}, err
		}
	}
	for _, v := range update {
		if err := c.Update(ctx, &v); err != nil {
			log.Error(err, "unable to update DNSRecord", "name", v.Name)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func diffRecords(got, want dnsv1alpha1.DNSRecordList) (add, update, del []dnsv1alpha1.DNSRecord) {
	for _, v := range want.Items {
		found := false
		for _, vv := range got.Items {
			if v.Name == vv.Name {
				found = true
				if !reflect.DeepEqual(v.Spec, vv.Spec) ||
					!reflect.DeepEqual(v.Annotations, vv.Annotations) ||
					!reflect.DeepEqual(v.OwnerReferences, vv.OwnerReferences) {
					v.SetResourceVersion(vv.GetResourceVersion())
					update = append(update, v)
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

func extractValue(apiVersion string, kind string) client.IndexerFunc {
	return func(rawObj client.Object) []string {
		// grab the owner object
		owner := metav1.GetControllerOf(rawObj)
		if owner == nil {
			return nil
		}

		if owner.APIVersion != apiVersion || owner.Kind != kind {
			return nil
		}

		return []string{owner.Name}
	}
}
