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
	Active *bool        `json:"active,omitempty"`
	A      *ARecord     `json:"a,omitempty"`
	CNAME  *CNAMERecord `json:"cname,omitempty"`
	TXT    *TXTRecord   `json:"txt,omitempty"`
	SRV    *SRVRecord   `json:"srv,omitempty"`
	MX     *MXRecord    `json:"mx,omitempty"`
	// Raw is an RFC 1035 style record string that github.com/miekg/dns will try to parse
	// +optional
	Raw string `json:"raw,omitempty"`
}

// DNSRecordStatus defines the observed state of DNSRecord
type DNSRecordStatus struct {
	Record   string `json:"record,omitempty"`
	Active   *bool  `json:"active,omitempty"`
	Provider string `json:"provider,omitempty"`
	ID       string `json:"id,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=dnsrecord,shortName=records;record;dns
// +kubebuilder:printcolumn:name="Active",type=boolean,JSONPath=`.status.active`
// +kubebuilder:printcolumn:name="Record",type=string,JSONPath=`.status.record`

// DNSRecord is the Schema for the dnsrecord API
type DNSRecord struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DNSRecordSpec   `json:"spec,omitempty"`
	Status DNSRecordStatus `json:"status,omitempty"`
}

type CNAMERecord struct {
	Name string `json:"name"`
	// +optional
	Class uint16 `json:"class,omitempty"`
	// +optional
	Ttl    uint32 `json:"ttl"`
	Target string `json:"target"`
}

type ARecord struct {
	Name string `json:"name"`
	// +optional
	Class uint16 `json:"class,omitempty"`
	// +optional
	Ttl uint32 `json:"ttl"`
	// TODO(adphi): support service, e.g. default/kubernetes
	Target string `json:"target,omitempty"`
}

type TXTRecord struct {
	Name string `json:"name"`
	// +optional
	Class uint16 `json:"class,omitempty"`
	// +optional
	Ttl     uint32   `json:"ttl"`
	Targets []string `json:"targets,omitempty"`
}

type SRVRecord struct {
	Name string `json:"name"`
	// +optional
	Class uint16 `json:"class,omitempty"`
	// +optional
	Ttl uint32 `json:"ttl"`
	// +optional
	Priority uint16 `json:"priority,omitempty"`
	// +optional
	Weight uint16 `json:"weight,omitempty"`
	Port   uint16 `json:"port,omitempty"`
	Target string `json:"target,omitempty"`
}

type MXRecord struct {
	Name string `json:"name"`
	// +optional
	Class uint16 `json:"class,omitempty"`
	// +optional
	Ttl uint32 `json:"ttl"`
	// +optional
	Preference uint16 `json:"preference,omitempty"`
	Target     string `json:"target,omitempty"`
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
