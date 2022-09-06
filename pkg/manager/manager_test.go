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

package manager

import (
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"gopkg.in/fsnotify.v1"

	"sigs.k8s.io/cluster-api-provider-vsphere/pkg/context/fake"
)

const (
	username        = "abcd"
	updatedUsername = "efgh"
	password        = "pass"
	updatedPassword = "ssap"
)

func TestManager_FileWatch(t *testing.T) {
	g := NewWithT(t)
	//*******Why do we need ---
	contentFmt := `---
username: '%s'
password: '%s'
`
	t.Run("update username & password for CAPV credentials", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "creds")
		if err != nil {
			t.Fatal(err)
		}
		//***** add this here
		//t.Cleanup(func() { os.Remove(tmpFile.Name()) })
		managerOptsTest := &Options{
			// needs an object ref to be present
			CredentialsFile: tmpFile.Name(),
			Username:        username,
			Password:        password,
		}

		watch, err := InitializeWatch(fake.NewControllerManagerContext(), managerOptsTest)
		// Match initial credentials
		g.Expect(err).To(BeNil())
		g.Expect(managerOptsTest.Username).To(Equal(username))
		g.Expect(managerOptsTest.Password).To(Equal(password))

		// Update the file and wait for watch to detect the change
		content := fmt.Sprintf(contentFmt, updatedUsername, updatedPassword)
		_, err = tmpFile.Write([]byte(content))
		g.Expect(err).To(BeNil())
		//*********see again how eventually works
		g.Eventually(func() bool {
			return managerOptsTest.Username == updatedUsername && managerOptsTest.Password == updatedPassword
		}, 10*time.Second).Should(BeTrue())
		//watch is passed as a parameter
		//see this image to understand how it works
		//https://miro.medium.com/max/1400/1*GpuBW-PeJt4LwumKJwUwGQ.png
		defer func(watch *fsnotify.Watcher) {
			_ = watch.Close()
		}(watch)
	})

	t.Run("send an error on watch error channel", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "creds")
		if err != nil {
			t.Fatal(err)
		}

		managerOptsTest := &Options{
			// needs an object ref to be present
			CredentialsFile: tmpFile.Name(),
			Username:        username,
			Password:        password,
		}
		watch, err := InitializeWatch(fake.NewControllerManagerContext(), managerOptsTest)
		// Match initial credentials
		g.Expect(err).To(BeNil())
		g.Expect(managerOptsTest.Username).To(Equal(username))
		g.Expect(managerOptsTest.Password).To(Equal(password))

		t.Log("sending an error on the channel")
		watch.Errors <- errors.Errorf("force failure")

		// Update the file and wait for watch to detect the change
		//****** why do we need this if watch errors are already supplied
		content := fmt.Sprintf(contentFmt, updatedUsername, updatedPassword)
		if _, err := tmpFile.Write([]byte(content)); err != nil {
			fmt.Printf("failed to update credentials in the file err:%s", err.Error())
		}

		//***if we are adding failed scnario above
		// tehn why are we checking for updated password
		g.Eventually(func() bool {
			return managerOptsTest.Username == updatedUsername && managerOptsTest.Password == updatedPassword
		}, 10*time.Second).Should(BeTrue())

		defer func(watch *fsnotify.Watcher) {
			_ = watch.Close()
		}(watch)
	})

	t.Run("force fail the watch", func(t *testing.T) {
		_, err := os.CreateTemp("", "creds")
		if err != nil {
			t.Fatal(err)
		}
		managerOptsTest := &Options{
			// needs an object ref to be present
			CredentialsFile: "fail",
			Username:        username,
			Password:        password,
		}
		_, err = InitializeWatch(fake.NewControllerManagerContext(), managerOptsTest)
		// Match initial credentials
		g.Expect(err).NotTo(BeNil())
	})
}
