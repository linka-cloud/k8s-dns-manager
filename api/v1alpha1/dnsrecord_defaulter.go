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

func (in *DNSRecord) Default() {
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
