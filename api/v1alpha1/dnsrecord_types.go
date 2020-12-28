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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DNSRecordSpec defines the desired state of DNSRecord
type DNSRecordSpec struct {
	Name string `json:"name"`
	// +optional
	Class uint16 `json:"class,omitempty"`
	Ttl   uint32 `json:"ttl,omitempty"`
	// +optional
	Target     string `json:"target,omitempty"`
	*ARecord   `json:",inline,omitempty"`
	*TXTRecord `json:",inline,omitempty"`
	*SRVRecord `json:",inline,omitempty"`
	*MXRecord  `json:",inline,omitempty"`
	// +optional
	Raw string `json:"raw,omitempty"`
}

// DNSRecordStatus defines the observed state of DNSRecord
type DNSRecordStatus struct {
	Record string `json:"record,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=dnsrecord,shortName=records;record;dns
// +kubebuilder:printcolumn:name="Record",type=string,JSONPath=`.status.record`

// DNSRecord is the Schema for the dnsrecords API
type DNSRecord struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DNSRecordSpec   `json:"spec,omitempty"`
	Status DNSRecordStatus `json:"status,omitempty"`
}

type ARecord struct {
	// TODO(adphi): support service, e.g. default/kubernetes
	A string `json:"a,omitempty"`
}

type TXTRecord struct {
	Txt []string `json:"txt,omitempty"`
}

type SRVRecord struct {
	// +optional
	Priority uint16 `json:"priority,omitempty"`
	// +optional
	Weight uint16 `json:"weight,omitempty"`
	Port   uint16 `json:"port,omitempty"`
	// Target   string `json:"target,omitempty"`
}

type MXRecord struct {
	// +optional
	Preference uint16 `json:"preference,omitempty"`
	Mx         string `json:"mx,omitempty"`
}

// +kubebuilder:object:root=true

// DNSRecordList contains a list of DNSRecord
type DNSRecordList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DNSRecord `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DNSRecord{}, &DNSRecordList{})
}
