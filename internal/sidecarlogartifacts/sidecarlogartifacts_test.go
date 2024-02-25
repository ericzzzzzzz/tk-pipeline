/*
Copyright 2024 The Tekton Authors

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

package sidecarlogartifacts

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/test/diff"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
)

func TestLookForArtifacts_WaitForFiles(t *testing.T) {
	tests := []struct {
		desc       string
		runDirMode os.FileMode
		stepDir    string
		outFile    string
	}{
		{
			desc:       "out.err file exist, no err",
			runDirMode: 0o755,
			stepDir:    "first",
			outFile:    "out.err",
		},
		{
			desc:       "out file exist, no err",
			runDirMode: 0o755,
			stepDir:    "first",
			outFile:    "out",
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			dir := t.TempDir()
			_ = os.Chmod(dir, tc.runDirMode)
			if tc.stepDir != "" {
				_ = os.MkdirAll(filepath.Join(dir, tc.stepDir), os.ModePerm)

				outFile := filepath.Join(dir, tc.stepDir, tc.outFile)
				_, err := os.Create(outFile)
				if err != nil {
					t.Fatalf("failed to create file %v", err)
				}
			}
			_, err := LookForArtifacts([]string{}, dir)
			if err != nil {
				t.Fatalf("failed to look for artifacts %v", err)
			}
		})
	}
}

func TestLookForArtifacts(t *testing.T) {
	base := basicArtifacts()
	var modified = base.DeepCopy()
	modified.Outputs[0].Name = "tests"
	type Arg struct {
		stepName      string
		artifacts     *v1.Artifacts
		customContent []byte
	}
	tests := []struct {
		desc     string
		wantErr  bool
		args     []Arg
		expected SidecarArtifacts
	}{
		{
			desc:     "one step produces artifacts, read success",
			args:     []Arg{{stepName: "first", artifacts: &base}},
			expected: map[string]v1.Artifacts{"first": base},
		}, {
			desc:     "two step produce artifacts, read success",
			args:     []Arg{{stepName: "first", artifacts: &base}, {stepName: "second", artifacts: modified}},
			expected: map[string]v1.Artifacts{"first": base, "second": *modified},
		},
		{
			desc:     "one step produces artifacts,  one step does not, read success",
			args:     []Arg{{stepName: "first", artifacts: &base}, {stepName: "second"}},
			expected: map[string]v1.Artifacts{"first": base},
		},
		{
			desc:     "two step produces,  one read success, one not, error out and result is empty.",
			args:     []Arg{{stepName: "first", artifacts: &base}, {stepName: "second", artifacts: modified, customContent: []byte("this is to break json")}},
			expected: map[string]v1.Artifacts{},
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			dir := t.TempDir()
			curStepDir := stepDir
			stepDir = dir
			t.Cleanup(func() {
				stepDir = curStepDir
			})

			var names []string
			for _, arg := range tc.args {
				names = append(names, arg.stepName)
				if err := os.MkdirAll(filepath.Join(dir, arg.stepName, "artifacts"), os.ModePerm); err != nil {
					t.Errorf("failed to create artifacts folder, err: %v", err)
				}
				if _, err := os.Create(filepath.Join(dir, arg.stepName, "out")); err != nil {
					t.Errorf("failed to file, err: %v", err)
				}
				if arg.artifacts != nil {
					if err := writeArtifacts(filepath.Join(dir, arg.stepName, "artifacts", "provenance.json"), arg.artifacts); err != nil {
						t.Errorf("failed to write artifacts to provenance.json, err: %v", err)
					}
				}
				if arg.customContent != nil {
					if err := os.WriteFile(filepath.Join(dir, arg.stepName, "artifacts", "provenance.json"), arg.customContent, os.ModePerm); err != nil {
						t.Errorf("failed to write customContent to provenance.json, err: %v", err)
					}
				}
			}
			got, err := LookForArtifacts(names, dir)
			if (err != nil) != tc.wantErr {
				t.Errorf("error checking failed, wantErr: %v, got: %v", tc.wantErr, err)
			}
			if d := cmp.Diff(tc.expected, got); d != "" {
				t.Errorf(diff.PrintWantGot(d))
			}
		})
	}
}

func TestGetArtifactsFromSidecarLogs(t *testing.T) {
	for _, c := range []struct {
		desc      string
		podPhase  corev1.PodPhase
		wantError bool
	}{{
		desc:      "pod pending to start",
		podPhase:  corev1.PodPending,
		wantError: false,
	}, {
		desc:      "pod running extract logs",
		podPhase:  corev1.PodRunning,
		wantError: true,
	}} {
		t.Run(c.desc, func(t *testing.T) {
			ctx := context.Background()
			clientset := fakekubeclientset.NewSimpleClientset()
			pod := &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod",
					Namespace: "foo",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "container",
							Image: "image",
						},
					},
				},
				Status: corev1.PodStatus{
					Phase: c.podPhase,
				},
			}
			pod, err := clientset.CoreV1().Pods(pod.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
			if err != nil {
				t.Errorf("Error occurred while creating pod %s: %s", pod.Name, err.Error())
			}

			// Fake logs are not formatted properly so there will be an error
			_, err = GetArtifactsFromSidecarLogs(ctx, clientset, "foo", "pod", "container", pod.Status.Phase)
			if err != nil && !c.wantError {
				t.Fatalf("did not expect an error but got: %v", err)
			}
			if c.wantError && err == nil {
				t.Fatal("expected to get an error but did not")
			}
		})
	}
}

func writeArtifacts(path string, artifacts *v1.Artifacts) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	res := json.NewEncoder(f).Encode(artifacts)
	return res
}

func basicArtifacts() v1.Artifacts {
	data := `{
            "inputs":[
              {
                "name":"inputs",
                "values":[
                  {
                    "uri":"pkg:example.github.com/inputs",
                    "digest":{
                      "sha256":"b35cacccfdb1e24dc497d15d553891345fd155713ffe647c281c583269eaaae0"
                    }
                  }
                ]
              }
            ],
            "outputs":[
              {
                "name":"image",
                "values":[
                  {
                    "uri":"pkg:github/package-url/purl-spec@244fd47e07d1004f0aed9c",
                    "digest":{
                      "sha256":"df85b9e3983fe2ce20ef76ad675ecf435cc99fc9350adc54fa230bae8c32ce48",
                      "sha1":"95588b8f34c31eb7d62c92aaa4e6506639b06ef2"
                    }
                  }
                ]
              }
            ]
          }
`
	var ars v1.Artifacts
	err := json.Unmarshal([]byte(data), &ars)
	if err != nil {
		panic(err)
	}
	return ars
}
