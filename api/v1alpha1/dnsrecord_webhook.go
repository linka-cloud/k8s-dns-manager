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

	"go.linka.cloud/k8s/dns/pkg/ptr"
)

// log is for logging in this package.
var dnsrecordlog = logf.Log.WithName("dnsrecord-resource")

func (r *DNSRecord) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-dns-linka-cloud-v1alpha1-dnsrecord,mutating=true,failurePolicy=fail,groups=dns.linka.cloud,resources=dnsrecord,verbs=create;update,versions=v1alpha1,name=mdnsrecord.kb.io,admissionReviewVersions=v1,sideEffects=None

var _ webhook.Defaulter = &DNSRecord{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (in *DNSRecord) Default() {
	if in.Spec.Active == nil {
		in.Spec.Active = ptr.Bool(true)
	}
	switch {
	case in.Spec.A != nil:
		if in.Spec.A.Class == 0 {
			in.Spec.A.Class = 1
		}
		if in.Spec.A.Ttl == 0 {
			in.Spec.A.Ttl = 3600
		}
	case in.Spec.CNAME != nil:
		if in.Spec.CNAME.Class == 0 {
			in.Spec.CNAME.Class = 1
		}
		if in.Spec.CNAME.Ttl == 0 {
			in.Spec.CNAME.Ttl = 3600
		}
	case in.Spec.TXT != nil:
		if in.Spec.TXT.Class == 0 {
			in.Spec.TXT.Class = 1
		}
		if in.Spec.TXT.Ttl == 0 {
			in.Spec.TXT.Ttl = 3600
		}
	case in.Spec.SRV != nil:
		if in.Spec.SRV.Class == 0 {
			in.Spec.SRV.Class = 1
		}
		if in.Spec.SRV.Ttl == 0 {
			in.Spec.SRV.Ttl = 3600
		}
		if in.Spec.SRV.Weight == 0 {
			in.Spec.SRV.Weight = 1
		}
	case in.Spec.MX != nil:
		if in.Spec.MX.Class == 0 {
			in.Spec.MX.Class = 1
		}
		if in.Spec.MX.Ttl == 0 {
			in.Spec.MX.Ttl = 3600
		}
		if in.Spec.MX.Preference == 0 {
			in.Spec.MX.Preference = 10
		}
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-dns-linka-cloud-v1alpha1-dnsrecord,mutating=false,failurePolicy=fail,groups=dns.linka.cloud,resources=dnsrecord,versions=v1alpha1,name=vdnsrecord.kb.io,admissionReviewVersions=v1,sideEffects=None

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
	switch {
	case r.Spec.A != nil:
		errs = append(errs, r.Spec.A.validate()...)
	case r.Spec.CNAME != nil:
		errs = append(errs, r.Spec.CNAME.validate()...)
	case r.Spec.TXT != nil:
		errs = append(errs, r.Spec.TXT.validate()...)
	case r.Spec.SRV != nil:
		errs = append(errs, r.Spec.SRV.validate()...)
	case r.Spec.MX != nil:
		errs = append(errs, r.Spec.MX.validate()...)
	case r.Spec.Raw != "":
		rr, err := dns.NewRR(r.Spec.Raw)
		if err != nil || rr == nil {
			errs = append(errs, field.Invalid(field.NewPath("spec").Child("raw"), r.Spec.Raw, "failed to parse raw record"))
		}
	default:
		errs = append(errs, field.Invalid(field.NewPath("spec"), r.Spec, "neither a A, CNAME, TXT, SRV, MX or RAW record"))
	}
	if len(errs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(schema.GroupKind{Group: GroupVersion.Group, Kind: r.Kind}, r.Name, errs)
}

func (r *ARecord) validate() (errs field.ErrorList) {
	if !strings.HasSuffix(r.Name, ".") {
		errs = append(errs, field.Invalid(field.NewPath("spec").Child("a").Child("name"), r.Name, "name must be an absolute dns name (should ends with a dot)"))
	}
	ip := net.ParseIP(r.Target)
	if ip == nil || r.Target == "" {
		errs = append(errs, field.Invalid(field.NewPath("spec").Child("a").Child("target"), r.Target, "A Record: target must be a valid ip address"))
	}
	return
}
func (r *CNAMERecord) validate() (errs field.ErrorList) {
	if !strings.HasSuffix(r.Name, ".") {
		errs = append(errs, field.Invalid(field.NewPath("spec").Child("cname").Child("name"), r.Name, "name must be an absolute dns name (should ends with a dot)"))
	}
	if r.Target == "" {
		errs = append(errs, field.Invalid(field.NewPath("spec").Child("cname").Child("target"), r.Target, "SRV record: target is required"))
	}
	if !strings.HasSuffix(r.Target, ".") {
		errs = append(errs, field.Invalid(field.NewPath("spec").Child("cname").Child("target"), r.Target, "target must be an absolute dns name (should ends with a dot)"))
	}
	return
}
func (r *TXTRecord) validate() (errs field.ErrorList) {
	if !strings.HasSuffix(r.Name, ".") {
		errs = append(errs, field.Invalid(field.NewPath("spec").Child("txt").Child("name"), r.Name, "name must be an absolute dns name (should ends with a dot)"))
	}
	if len(r.Targets) == 0 {
		errs = append(errs, field.Invalid(field.NewPath("spec").Child("txt").Child("targets"), r.Targets, "TXT Record: target cannot be empty"))
	}
	return
}
func (r *SRVRecord) validate() (errs field.ErrorList) {
	if !strings.HasSuffix(r.Name, ".") {
		errs = append(errs, field.Invalid(field.NewPath("spec").Child("srv").Child("name"), r.Name, "must be an absolute dns name (should ends with a dot)"))
	}
	if r.Target == "" {
		errs = append(errs, field.Invalid(field.NewPath("spec").Child("srv").Child("target"), r.Target, "SRV record: target is required"))
	}
	if !strings.HasSuffix(r.Target, ".") {
		errs = append(errs, field.Invalid(field.NewPath("spec").Child("srv").Child("target"), r.Target, "must be an absolute dns name (should ends with a dot)"))
	}
	return
}
func (r *MXRecord) validate() (errs field.ErrorList) {
	if !strings.HasSuffix(r.Name, ".") {
		errs = append(errs, field.Invalid(field.NewPath("spec").Child("mx").Child("name"), r.Name, "must be an absolute dns name (should ends with a dot)"))
	}
	if r.Target == "" {
		errs = append(errs, field.Invalid(field.NewPath("spec").Child("mx").Child("target"), r.Target, "MX Record: target cannot be empty"))
	}
	if !strings.HasSuffix(r.Target, ".") {
		errs = append(errs, field.Invalid(field.NewPath("spec").Child("mx").Child("target"), r.Target, "must be an absolute dns name (should ends with a dot)"))
	}
	return
}
