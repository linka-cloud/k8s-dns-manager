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
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/miekg/dns"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dnsv1alpha1 "go.linka.cloud/k8s/dns/api/v1alpha1"
	"go.linka.cloud/k8s/dns/pkg/record"
	"go.linka.cloud/k8s/dns/pkg/recorder"
)

const (
	RecordFinalizer = "dns.linka.cloud/finalizer"
)

// DNSRecordReconciler reconciles a DNSRecord object
type DNSRecordReconciler struct {
	client.Client
	Log       logr.Logger
	Scheme    *runtime.Scheme
	recorder  recorder.Recorder
	dnsClient *dns.Client
}

// +kubebuilder:rbac:groups=dns.linka.cloud,resources=dnsrecords,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dns.linka.cloud,resources=dnsrecords/status,verbs=get;update;patch

func (r *DNSRecordReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("dnsrecord", req.NamespacedName)
	log.V(2).Info("new request")
	var rec dnsv1alpha1.DNSRecord
	if err := r.Get(ctx, req.NamespacedName, &rec); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch record")
		return ctrl.Result{}, err
	}

	// TODO(adphi): SOA and NS Records ?
	rec.Default()
	rr, err := record.ToRR(rec)
	if err != nil {
		r.recorder.Warn(&rec, "Error", err.Error())
		log.Error(err, "parse record")
		return ctrl.Result{}, err
	}

	if !rec.DeletionTimestamp.IsZero() {
		log.Info("record marked for deletion: deleting")
		// TODO(adphi): update SOA Record
		if ok := removeFinalizer(&rec); !ok {
			return ctrl.Result{}, nil
		}
		if err := r.Update(ctx, &rec); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if !hasFinalizer(rec) {
		log.Info("setting record finalizer")
		rec.Finalizers = append(rec.Finalizers, RecordFinalizer)
		if err := r.Update(ctx, &rec); err != nil {
			log.Error(err, "set finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if rec.Status.Record != rr.String() {
		log.Info("updating record status")
		rec.Status.Record = rr.String()
		if err := r.Status().Update(ctx, &rec); err != nil {
			log.Error(err, "update status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	ok, err := r.lookup(ctx, rr)
	if err != nil {
		log.Error(err, "lookup failed")
		return ctrl.Result{}, err
	}
	if rec.Status.Active != ok {
		log.Info("updating record status", "active", ok)
		rec.Status.Active = ok
		if err := r.Status().Update(ctx, &rec); err != nil {
			log.Error(err, "update status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	if rec.Spec.Active == nil {
		active := true
		rec.Spec.Active = &active
	}
	var state string
	if ok {
		state = "active"
	} else {
		state = "inactive"
	}
	if *rec.Spec.Active != ok {
		r.recorder.Warn(&rec, "Warning", fmt.Sprintf("record %s", state))
		log.Info("status does not match desired state", "desired", *rec.Spec.Active, "actual", ok)
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}
	r.recorder.Event(&rec, "Success", fmt.Sprintf("record %s", state))
	return ctrl.Result{}, nil
}

func (r *DNSRecordReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.dnsClient = new(dns.Client)
	r.recorder = recorder.New(mgr.GetEventRecorderFor("DNSRecord"))
	return ctrl.NewControllerManagedBy(mgr).
		For(&dnsv1alpha1.DNSRecord{}).
		Complete(r)
}

func (r *DNSRecordReconciler) lookup(ctx context.Context, rr dns.RR) (bool, error) {
	q := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:               dns.Id(),
			RecursionDesired: true,
		},
		Question: []dns.Question{{
			Name:   rr.Header().Name,
			Qtype:  rr.Header().Rrtype,
			Qclass: rr.Header().Class,
		}},
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	res, _, err := r.dnsClient.ExchangeContext(ctx, q, "127.0.0.1:53")
	if err != nil {
		return false, err
	}
	if len(res.Answer) == 0 {
		return false, nil
	}
	// TODO(adphi): check response
	for _, v := range res.Answer {
		if dns.IsDuplicate(v, rr) {
			return true, nil
		}
	}
	return false, nil
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
