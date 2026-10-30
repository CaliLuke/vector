package main

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

type cleanupResource struct {
	closeErr error
	closes   int
	onClose  func()
}

func (resource *cleanupResource) Close() error {
	resource.closes++
	if resource.onClose != nil {
		resource.onClose()
	}
	return resource.closeErr
}

func TestResourceCleanupLeavesSuccessfulRunSuccessful(t *testing.T) {
	resource := &cleanupResource{}
	var resultErr error
	closeResource(&resultErr, "state cache", resource)
	if resultErr != nil || resource.closes != 1 {
		t.Fatalf("error=%v closes=%d", resultErr, resource.closes)
	}
}

func TestResourceCleanupPreservesOriginalErrorWhenCloseSucceeds(t *testing.T) {
	startupErr := errors.New("startup failed")
	resultErr := startupErr
	resource := &cleanupResource{}
	closeResource(&resultErr, "state cache", resource)
	if !errors.Is(resultErr, startupErr) || resource.closes != 1 {
		t.Fatalf("error=%v closes=%d", resultErr, resource.closes)
	}
}

func TestResourceCleanupFailureMakesSuccessfulRunFail(t *testing.T) {
	cleanupErr := errors.New("connection close failed")
	resource := &cleanupResource{closeErr: cleanupErr}
	var resultErr error
	closeResource(&resultErr, "state cache", resource)
	if !errors.Is(resultErr, cleanupErr) || resultErr.Error() != "close state cache: connection close failed" || resource.closes != 1 {
		t.Fatalf("error=%v closes=%d", resultErr, resource.closes)
	}
}

func TestResourceCleanupRetainsStartupAndCloseFailures(t *testing.T) {
	startupErr := errors.New("startup failed")
	cleanupErr := errors.New("connection close failed")
	resultErr := startupErr
	resource := &cleanupResource{closeErr: cleanupErr}
	closeResource(&resultErr, "state cache", resource)
	if !errors.Is(resultErr, startupErr) || !errors.Is(resultErr, cleanupErr) || !strings.Contains(resultErr.Error(), "close state cache") || resource.closes != 1 {
		t.Fatalf("error=%v closes=%d", resultErr, resource.closes)
	}
}

func TestResourceCleanupContinuesAfterFailuresInReverseAcquisitionOrder(t *testing.T) {
	firstErr := errors.New("first cache close failed")
	lastErr := errors.New("last cache close failed")
	serveErr := errors.New("serve failed")
	var order []string
	first := &cleanupResource{closeErr: firstErr, onClose: func() { order = append(order, "first") }}
	middle := &cleanupResource{onClose: func() { order = append(order, "middle") }}
	last := &cleanupResource{closeErr: lastErr, onClose: func() { order = append(order, "last") }}
	resultErr := func() (resultErr error) {
		defer closeResource(&resultErr, "first cache", first)
		defer closeResource(&resultErr, "middle cache", middle)
		defer closeResource(&resultErr, "last cache", last)
		return serveErr
	}()
	if !errors.Is(resultErr, serveErr) || !errors.Is(resultErr, firstErr) || !errors.Is(resultErr, lastErr) {
		t.Fatalf("lost failure: %v", resultErr)
	}
	if !slices.Equal(order, []string{"last", "middle", "first"}) {
		t.Fatalf("cleanup order=%v", order)
	}
	if first.closes != 1 || middle.closes != 1 || last.closes != 1 {
		t.Fatalf("close counts=%d,%d,%d", first.closes, middle.closes, last.closes)
	}
}
