/*
Copyright 2022 The Kubernetes Authors.

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

package session

import (
	"strings"

	infrav1 "sigs.k8s.io/cluster-api-provider-vsphere/apis/v1beta1"
)

func (s Session) GetVersion() (infrav1.VCenterVersion, error) {
	version := s.ServiceContent.About.Version
	switch {
	case strings.HasPrefix(version, "6.7"):
		return infrav1.Version67, nil
	case strings.HasPrefix(version, "7.0"):
		return infrav1.Version70, nil
	case strings.HasPrefix(version, "8.0"):
		return infrav1.Version80, nil
	default:
		return "", unidentifiedVCenterVersion{version: version}
	}
}
