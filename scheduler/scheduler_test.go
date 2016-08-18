// Copyright 2016 Mirantis
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package scheduler

import (
	"testing"

	"github.com/Mirantis/k8s-AppController/mocks"
)

func TestBuildDependencyGraph(t *testing.T) {
	c := mocks.NewClient()
	c.ResourceDefinitionsInterface = mocks.NewResourceDefinitionClient("pod/ready-1", "pod/ready-2")
	c.DependenciesInterface = mocks.NewDependencyClient(
		mocks.Dependency{Parent: "pod/ready-1", Child: "pod/ready-2"})

	depGraphPtr, err := BuildDependencyGraph(c, nil)
	if err != nil {
		t.Error(err)
	}

	depGraph := *depGraphPtr

	if len(depGraph) != 2 {
		t.Errorf("Wrong length of dependency graph, expected %d, actual %d",
			2, len(depGraph))
	}

	sr, ok := depGraph["pod/ready-1"]

	if !ok {
		t.Errorf("Dependency for '%s' not found in dependency graph", "pod/ready-1")
	}

	if sr.Key() != "pod/ready-1" {
		t.Errorf("Wrong scheduled resource key, expected '%s', actual '%s'",
			"pod/ready-1", sr.Key())
	}

	if len(sr.Requires) != 0 {
		t.Errorf("Wrong length of 'Requires' for scheduled resource '%s', expected %d, actual %d",
			sr.Key(), 0, len(sr.Requires))
	}

	if len(sr.RequiredBy) != 1 {
		t.Errorf("Wrong length of 'RequiredBy' for scheduled resource '%s', expected %d, actual %d",
			sr.Key(), 1, len(sr.Requires))
	}

	if sr.RequiredBy[0].Key() != "pod/ready-2" {
		t.Errorf("Wrong value of 'RequiredBy' for scheduled resource '%s', expected '%s', actual '%s'",
			sr.Key(), "pod/ready-2", sr.RequiredBy[0].Key())
	}

	sr, ok = (depGraph)["pod/ready-2"]

	if !ok {
		t.Errorf("Dependency for '%s' not found in dependency graph", "pod/ready-2")
	}

	if sr.Key() != "pod/ready-2" {
		t.Errorf("Wrong scheduled resource key, expected '%s', actual '%s'",
			"pod/ready-2", sr.Key())
	}

	if len(sr.Requires) != 1 {
		t.Errorf("Wrong length of 'Requires' for scheduled resource '%s', expected %d, actual %d",
			sr.Key(), 1, len(sr.Requires))
	}

	if sr.Requires[0].Key() != "pod/ready-1" {
		t.Errorf("Wrong value of 'Requires' for scheduled resource '%s', expected '%s', actual '%s'",
			sr.Key(), "pod/ready-1", sr.Requires[0].Key())
	}

	if len(sr.RequiredBy) != 0 {
		t.Errorf("Wrong length of 'RequiredBy' for scheduled resource '%s', expected %d, actual %d",
			sr.Key(), 0, len(sr.Requires))
	}
}

func TestIsBlocked(t *testing.T) {
	one := &ScheduledResource{Status: Init}

	if one.IsBlocked() {
		t.Errorf("Scheduled resource is blocked but it must not")
	}

	two := &ScheduledResource{Status: Ready}
	three := &ScheduledResource{Status: Ready}

	one.Requires = []*ScheduledResource{two, three}

	if one.IsBlocked() {
		t.Errorf("Scheduled resource is blocked but it must not")
	}

	one.Requires[0].Status = Creating

	if !one.IsBlocked() {
		t.Errorf("Scheduled resource is not blocked but it must be")
	}
}

func TestDetectCyclesAcyclic(t *testing.T) {
	c := mocks.NewClient()
	c.ResourceDefinitionsInterface = mocks.NewResourceDefinitionClient("pod/ready-1", "pod/ready-2")
	c.DependenciesInterface = mocks.NewDependencyClient(
		mocks.Dependency{Parent: "pod/ready-1", Child: "pod/ready-2"})

	depGraphPtr, _ := BuildDependencyGraph(c, nil)

	cycles := DetectCycles(*depGraphPtr)

	if len(cycles) != 0 {
		t.Errorf("Cycles detected in an acyclic graph: %v", cycles)
	}
}

func TestDetectCyclesSimpleCycle(t *testing.T) {
	c := mocks.NewClient()
	c.ResourceDefinitionsInterface = mocks.NewResourceDefinitionClient("pod/ready-1", "pod/ready-2")
	c.DependenciesInterface = mocks.NewDependencyClient(
		mocks.Dependency{Parent: "pod/ready-1", Child: "pod/ready-2"},
		mocks.Dependency{Parent: "pod/ready-2", Child: "pod/ready-1"})

	depGraphPtr, _ := BuildDependencyGraph(c, nil)

	cycles := DetectCycles(*depGraphPtr)

	if len(cycles) != 1 {
		t.Errorf("Expected %d cycles, got %d", 1, len(cycles))
		return
	}
}

func TestDetectCyclesSelf(t *testing.T) {
	c := mocks.NewClient()
	c.ResourceDefinitionsInterface = mocks.NewResourceDefinitionClient("pod/ready-1")
	c.DependenciesInterface = mocks.NewDependencyClient(
		mocks.Dependency{Parent: "pod/ready-1", Child: "pod/ready-1"})

	depGraphPtr, _ := BuildDependencyGraph(c, nil)

	cycles := DetectCycles(*depGraphPtr)

	if len(cycles) != 1 {
		t.Errorf("Expected %d cycles, got %d", 1, len(cycles))
		return
	}

	if len(cycles[0]) != 2 {
		t.Errorf("Expected cycle length to be %d, got %d", 2, len(cycles[0]))
	}

	if cycles[0][0].Key() != "pod/ready-1" {
		t.Errorf("Expected cycle node key to be %s, got %s", "pod/ready-1", cycles[0][0].Key())
	}
	if cycles[0][1].Key() != "pod/ready-1" {
		t.Errorf("Expected cycle node key to be %s, got %s", "pod/ready-1", cycles[0][0].Key())
	}
}

func TestDetectCyclesLongCycle(t *testing.T) {
	c := mocks.NewClient()
	c.ResourceDefinitionsInterface = mocks.NewResourceDefinitionClient("pod/1", "pod/2", "pod/3", "pod/4", "pod/5")
	c.DependenciesInterface = mocks.NewDependencyClient(
		mocks.Dependency{Parent: "pod/1", Child: "pod/2"},
		mocks.Dependency{Parent: "pod/2", Child: "pod/3"},
		mocks.Dependency{Parent: "pod/3", Child: "pod/4"},
		mocks.Dependency{Parent: "pod/4", Child: "pod/1"},
	)

	depGraphPtr, _ := BuildDependencyGraph(c, nil)

	cycles := DetectCycles(*depGraphPtr)

	if len(cycles) != 1 {
		t.Errorf("Expected %d cycles, got %d", 1, len(cycles))
		return
	}

	if len(cycles[0]) != 4 {
		t.Errorf("Expected cycle length to be %d, got %d", 4, len(cycles[0]))
	}
}

func TestDetectCyclesComplex(t *testing.T) {
	c := mocks.NewClient()
	c.ResourceDefinitionsInterface = mocks.NewResourceDefinitionClient("pod/1", "pod/2", "pod/3", "pod/4", "pod/5")
	c.DependenciesInterface = mocks.NewDependencyClient(
		mocks.Dependency{Parent: "pod/1", Child: "pod/2"},
		mocks.Dependency{Parent: "pod/2", Child: "pod/3"},
		mocks.Dependency{Parent: "pod/3", Child: "pod/1"},
		mocks.Dependency{Parent: "pod/4", Child: "pod/1"},
		mocks.Dependency{Parent: "pod/1", Child: "pod/5"},
		mocks.Dependency{Parent: "pod/5", Child: "pod/4"},
	)

	depGraphPtr, _ := BuildDependencyGraph(c, nil)

	cycles := DetectCycles(*depGraphPtr)

	if len(cycles) != 1 {
		t.Errorf("Expected %d cycles, got %d", 1, len(cycles))
		return
	}

	if len(cycles[0]) != 5 {
		t.Errorf("Expected cycle length to be %d, got %d", 5, len(cycles[0]))
	}
}

func TestDetectCyclesMultiple(t *testing.T) {
	c := mocks.NewClient()
	c.ResourceDefinitionsInterface = mocks.NewResourceDefinitionClient(
		"pod/1", "pod/2", "pod/3", "pod/4", "pod/5", "pod/6", "pod/7")
	c.DependenciesInterface = mocks.NewDependencyClient(
		mocks.Dependency{Parent: "pod/1", Child: "pod/2"},
		mocks.Dependency{Parent: "pod/2", Child: "pod/3"},
		mocks.Dependency{Parent: "pod/3", Child: "pod/4"},
		mocks.Dependency{Parent: "pod/4", Child: "pod/2"},
		mocks.Dependency{Parent: "pod/1", Child: "pod/5"},
		mocks.Dependency{Parent: "pod/5", Child: "pod/6"},
		mocks.Dependency{Parent: "pod/6", Child: "pod/7"},
		mocks.Dependency{Parent: "pod/7", Child: "pod/5"},
	)

	depGraphPtr, _ := BuildDependencyGraph(c, nil)

	cycles := DetectCycles(*depGraphPtr)

	if len(cycles) != 2 {
		t.Errorf("Expected %d cycles, got %d", 2, len(cycles))
		return
	}

	if len(cycles[0]) != 3 {
		t.Errorf("Expected cycle length to be %d, got %d", 3, len(cycles[0]))
	}

	if len(cycles[1]) != 3 {
		t.Errorf("Expected cycle length to be %d, got %d", 3, len(cycles[1]))
	}
}