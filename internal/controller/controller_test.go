package controller

import (
	"github.com/stretchr/testify/assert"
	"kube-proxless/internal/cluster/fake"
	"kube-proxless/internal/config"
	"kube-proxless/internal/memory"
	"kube-proxless/internal/utils"
	"testing"
	"time"
)

func TestController_GetRouteByDomainFromMemory(t *testing.T) {
	c := NewController(memory.NewMemoryMap(), nil, nil)

	// error - memory is empty
	_, err := c.GetRouteByDomainFromMemory("mock.io")
	assert.Error(t, err)

	// add route in memory and test again
	assert.NoError(
		t,
		c.memory.UpsertMemoryMap(
			"mock-id", "mock-svc", "", "mock-deploy", "mock-ns", []string{"mock.io"}))

	r, err := c.GetRouteByDomainFromMemory("mock.io")
	assert.NoError(t, err)

	if r.GetId() != "mock-id" || r.GetService() != "mock-svc" || r.GetDeployment() != "mock-deploy" ||
		r.GetNamespace() != "mock-ns" || !utils.CompareUnorderedArray(r.GetDomains(), []string{"mock.io"}) {
		t.Errorf("GetRouteByDomainFromMemory('mock.io') = %v; route in memory does not match", r)
	}
}

func TestController_UpdateLastUseMemory(t *testing.T) {
	c := NewController(memory.NewMemoryMap(), nil, nil)

	// error - memory is empty
	assert.Error(t, c.UpdateLastUsedInMemory("mock.io"))

	// add route in memory and test again
	assert.NoError(
		t,
		c.memory.UpsertMemoryMap(
			"mock-id", "mock-svc", "", "mock-deploy", "mock-ns", []string{"mock.io"}))

	routeBefore, err := c.GetRouteByDomainFromMemory("mock.io")
	assert.NoError(t, err)
	timeBefore := routeBefore.GetLastUsed()

	time.Sleep(time.Second)

	assert.NoError(t, c.UpdateLastUsedInMemory("mock.io"))

	routeAfter, err := c.GetRouteByDomainFromMemory("mock.io")
	assert.NoError(t, err)

	if !routeAfter.GetLastUsed().After(timeBefore) {
		t.Errorf("UpdateLastUsedInMemory(); routeAfter = %s <= routeBefore = %s",
			routeAfter.GetLastUsed(), timeBefore)
	}
}

func TestController_ScaleUpDeployment(t *testing.T) {
	c := NewController(memory.NewMemoryMap(), fake.NewCluster(), nil)

	// check the implemention of the fake client to understand the test

	assert.NoError(t, c.ScaleUpDeployment("mock-deploy", "mock-ns"))

	assert.Error(t, c.ScaleUpDeployment("deploy", "ns"))
}

func TestController_scaleDownDeployments(t *testing.T) {
	c := NewController(memory.NewMemoryMap(), fake.NewCluster(), nil)

	// error - memory is empty / route not found
	helper_assertAtLeastOneError(t, scaleDownDeployments(c))

	// add route in memory and test again
	assert.NoError(
		t,
		c.memory.UpsertMemoryMap(
			"mock-id", "mock-svc", "", "mock-deploy", "mock-ns", []string{"mock.io"}))

	helper_assertNoError(t, scaleDownDeployments(c))
}

func TestController_RunDownScaler(t *testing.T) {
	c := NewController(memory.NewMemoryMap(), fake.NewCluster(), nil)

	// TODO check how we wanna deal with closing the channel and stopping the routine
	// We could use a context https://github.com/kubernetes/client-go/blob/master/examples/fake-client/main_test.go
	// but not sure if it is worth it
	go c.RunDownScaler(1)

	// make sure there is no panic when memory empty
	time.Sleep(1 * time.Second)

	// add route in memorymemory and test again
	assert.NoError(
		t,
		c.memory.UpsertMemoryMap(
			"mock-id", "mock-svc", "", "mock-deploy", "mock-ns", []string{"mock.io"}))

	// make sure there is no panic when memory had data
	time.Sleep(1 * time.Second)
}

func TestController_RunServicesEngine(t *testing.T) {
	c := NewController(memory.NewMemoryMap(), fake.NewCluster(), nil)

	// check the implemention of the fake client to understand the test
	config.NamespaceScope = "upsert"
	c.RunServicesEngine()

	_, err := c.memory.GetRouteByDomain("mock.io")
	assert.NoError(t, err)

	config.NamespaceScope = "delete"
	c.RunServicesEngine()

	_, err = c.memory.GetRouteByDomain("mock.io")
	assert.Error(t, err)

}
