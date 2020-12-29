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

package v1alpha1

import (
	"net"
	"strings"

	"github.com/miekg/dns"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var dnsrecordlog = logf.Log.WithName("dnsrecord-resource")

func (r *DNSRecord) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-dns-linka-cloud-v1alpha1-dnsrecord,mutating=true,failurePolicy=fail,groups=dns.linka.cloud,resources=dnsrecords,verbs=create;update,versions=v1alpha1,name=mdnsrecord.kb.io

var _ webhook.Defaulter = &DNSRecord{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *DNSRecord) Default() {
	dnsrecordlog.Info("default", "name", in.Name)

	if in.Spec.Class == 0 {
		in.Spec.Class = 1
	}
	if in.Spec.Ttl == 0 {
		in.Spec.Ttl = 3600
	}
	switch {
	case in.Spec.SRVRecord != nil:
		if in.Spec.SRVRecord.Weight == 0 {
			in.Spec.SRVRecord.Weight = 1
		}
	case in.Spec.MXRecord != nil:
		if in.Spec.MXRecord.Preference == 0 {
			in.Spec.MXRecord.Preference = 10
		}
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-dns-linka-cloud-v1alpha1-dnsrecord,mutating=false,failurePolicy=fail,groups=dns.linka.cloud,resources=dnsrecords,versions=v1alpha1,name=vdnsrecord.kb.io

var _ webhook.Validator = &DNSRecord{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *DNSRecord) ValidateCreate() error {
	dnsrecordlog.Info("validate create", "name", r.Name)
	return r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *DNSRecord) ValidateUpdate(old runtime.Object) error {
	dnsrecordlog.Info("validate update", "name", r.Name)
	return r.validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *DNSRecord) ValidateDelete() error {
	dnsrecordlog.Info("validate delete", "name", r.Name)
	return r.validate()
}

func (r *DNSRecord) validate() error {
	var errs field.ErrorList
	if !strings.HasSuffix(r.Spec.Name, ".") {
		errs = append(errs, field.Invalid(field.NewPath("spec").Child("name"), r.Spec.Name, "must be an absolute dns name (should ends with a dot)"))
	}
	if r.Spec.SRVRecord != nil && r.Spec.Target == "" {
		errs = append(errs, field.Invalid(field.NewPath("spec").Child("target"), r.Spec.Target, "SRV record: target is required"))
	}
	if r.Spec.ARecord != nil {
		ip := net.ParseIP(r.Spec.ARecord.A)
		if ip == nil || r.Spec.ARecord.A == "" {
			errs = append(errs, field.Invalid(field.NewPath("spec").Child("a"), r.Spec.A, "A Record: a must be a valid ip address"))
		}
	}
	if r.Spec.TXTRecord != nil && len(r.Spec.TXTRecord.Txt) == 0 {
		errs = append(errs, field.Invalid(field.NewPath("spec").Child("txt"), r.Spec.Txt, "TXT Record: txt cannot be empty"))
	}
	if r.Spec.MXRecord != nil && r.Spec.MXRecord.Mx == "" {
		errs = append(errs, field.Invalid(field.NewPath("spec").Child("mx"), r.Spec.Txt, "MX Record: mx cannot be empty"))
	}
	if r.Spec.ARecord == nil && r.Spec.TXTRecord == nil && r.Spec.MXRecord == nil && r.Spec.Raw == "" && r.Spec.Target == "" {
		errs = append(errs, field.Invalid(field.NewPath("spec").Child("target"), r.Spec.Txt, "CNAME Record: target cannot be empty"))
	}
	if r.Spec.Raw != "" {
		rr, err := dns.NewRR(r.Spec.Raw)
		if err != nil || rr == nil {
			errs = append(errs, field.Invalid(field.NewPath("spec").Child("raw"), r.Spec.Txt, "Raw Record is invalid"))
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: r.Kind}, r.Name, errs)
}
