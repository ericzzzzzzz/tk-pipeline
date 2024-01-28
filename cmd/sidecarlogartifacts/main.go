/*
Copyright 2019 The Tekton Authors

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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/tektoncd/pipeline/internal/sidecarlogartifacts"
	"log"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
)

func main() {
	var stepNames string
	flag.StringVar(&stepNames, "step-names", "", "comma separated step names to expect from the steps running in the pod. eg. foo,bar,baz")
	flag.Parse()
	if stepNames == "" {
		log.Fatal("step-names were not provided")
	}
	names := strings.Split(stepNames, ",")
	artifacts, err := sidecarlogartifacts.LookForArtifacts(names, pipeline.StepsDir)
	if err != nil {
		log.Fatal(err)
	}
	marshal, err := json.Marshal(artifacts)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(marshal)
}
