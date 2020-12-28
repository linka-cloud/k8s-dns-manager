/*


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
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/go-logr/logr"
	"github.com/miekg/dns"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dnsv1alpha1 "go.linka.cloud/k8s/dns/api/v1alpha1"
	"go.linka.cloud/k8s/dns/pkg/recorder"
)

const (
	RecordFinalizer = "dns.linka.cloud/finalizer"
)

// DNSRecordReconciler reconciles a DNSRecord object
type DNSRecordReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	records  map[string]dns.RR
	mu       sync.Mutex
	recorder recorder.Recorder
}

// +kubebuilder:rbac:groups=dns.linka.cloud,resources=dnsrecords,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dns.linka.cloud,resources=dnsrecords/status,verbs=get;update;patch

func (r *DNSRecordReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	if r.records == nil {
		r.mu.Lock()
		r.records = make(map[string]dns.RR)
		r.mu.Unlock()
	}
	log := r.Log.WithValues("dnsrecord", req.NamespacedName)
	log.V(2).Info("new request")
	var record dnsv1alpha1.DNSRecord
	if err := r.Get(ctx, req.NamespacedName, &record); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch record")
		return ctrl.Result{}, err
	}

	record.Default()
	rr, err := toRR(record)
	if err != nil {
		r.recorder.Warn(&record, "Error", err.Error())
		log.Error(err, "parse record")
		return ctrl.Result{}, err
	}

	if !record.DeletionTimestamp.IsZero() {
		log.Info("record marked for deletion: deleting")
		r.mu.Lock()
		delete(r.records, fmt.Sprintf("%s:%d", rr.Header().Name, rr.Header().Rrtype))
		r.mu.Unlock()
		if ok := removeFinalizer(&record); !ok {
			return ctrl.Result{}, nil
		}
		if err := r.Update(ctx, &record); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if !hasFinalizer(record) {
		log.Info("setting record finalizer")
		record.Finalizers = append(record.Finalizers, RecordFinalizer)
		if err := r.Update(ctx, &record); err != nil {
			log.Error(err, "set finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if record.Status.Record != rr.String() {
		log.Info("updating record status")
		record.Status.Record = rr.String()
		if err := r.Status().Update(ctx, &record); err != nil {
			log.Error(err, "update status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	log.Info("storing record")
	r.mu.Lock()
	r.records[fmt.Sprintf("%s:%d", rr.Header().Name, rr.Header().Rrtype)] = rr
	r.mu.Unlock()
	r.recorder.Event(&record, "Success", "record reconciled")
	return ctrl.Result{}, nil
}

func (r *DNSRecordReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = recorder.New(mgr.GetEventRecorderFor("DNSRecord"))
	return ctrl.NewControllerManagedBy(mgr).
		For(&dnsv1alpha1.DNSRecord{}).
		Complete(r)
}

func toRR(r dnsv1alpha1.DNSRecord) (dns.RR, error) {
	h := dns.RR_Header{
		Name:  r.Spec.Name,
		Class: r.Spec.Class,
		Ttl:   r.Spec.Ttl,
	}
	switch {
	case r.Spec.ARecord != nil:
		h.Rrtype = dns.TypeA
		ip := net.ParseIP(r.Spec.ARecord.A)
		if ip == nil {
			return nil, fmt.Errorf("invalid ip: %s", r.Spec.ARecord.A)
		}
		return &dns.A{Hdr: h, A: ip}, nil
	case r.Spec.TXTRecord != nil:
		h.Rrtype = dns.TypeTXT
		if len(r.Spec.Txt) == 0 {
			return nil, errors.New("empty TXT record")
		}
		return &dns.TXT{Hdr: h, Txt: r.Spec.TXTRecord.Txt}, nil
	case r.Spec.SRVRecord != nil:
		h.Rrtype = dns.TypeSRV
		if r.Spec.Target == "" {
			return nil, errors.New("'target' is required for SRV Records")
		}
		return &dns.SRV{
			Hdr:      h,
			Priority: r.Spec.SRVRecord.Priority,
			Weight:   r.Spec.SRVRecord.Weight,
			Port:     r.Spec.SRVRecord.Port,
			Target:   r.Spec.Target,
		}, nil
	case r.Spec.MXRecord != nil:
		h.Rrtype = dns.TypeMX
		return &dns.MX{Hdr: h, Preference: r.Spec.MXRecord.Preference, Mx: r.Spec.MXRecord.Mx}, nil
	case r.Spec.Target != "":
		h.Rrtype = dns.TypeCNAME
		return &dns.CNAME{Hdr: h, Target: r.Spec.Target}, nil
	default:
		if r.Spec.Raw == "" {
			return nil, errors.New("unknown record type")
		}
		rr, err := dns.NewRR(r.Spec.Raw)
		if err != nil {
			return nil, err
		}
		if rr == nil {
			return nil, fmt.Errorf("invalid record: '%s'", r.Spec.Raw)
		}
		return rr, nil
	}
}

func hasFinalizer(r dnsv1alpha1.DNSRecord) bool {
	for _, v := range r.Finalizers {
		if v == RecordFinalizer {
			return true
		}
	}
	return false
}

func removeFinalizer(r *dnsv1alpha1.DNSRecord) bool {
	for i, v := range r.ObjectMeta.Finalizers {
		if v != RecordFinalizer {
			continue
		}
		if len(r.Finalizers) == 1 {
			r.Finalizers = nil
			return true
		}
		r.ObjectMeta.Finalizers = append(r.ObjectMeta.Finalizers[i:], r.ObjectMeta.Finalizers[i+1:]...)
		return true
	}
	return false
}
