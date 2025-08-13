package promise

import (
	"errors"
	"testing"
)

func TestMicrotaskPanicHandling(t *testing.T) {
	t.Run("Panic in Then callback", func(t *testing.T) {
		promise := New(func(resolve func(string), reject func(error)) {
			resolve("success")
		})

		// Add a Then callback that will panic
		nextPromise := promise.Then(func(value string) any {
			panic("intentional panic in Then callback")
		}, nil)

		// The next promise should be rejected due to panic
		_, err := nextPromise.Await()
		if err == nil {
			t.Error("Expected error due to panic, but got none")
		}
		if err.Error() != "panic occurred in callback" {
			t.Errorf("Expected panic error, but got: %v", err)
		}
	})

	t.Run("Panic in Catch callback", func(t *testing.T) {
		promise := New(func(resolve func(string), reject func(error)) {
			reject(errors.New("original error"))
		})

		// Add a Catch callback that will panic
		nextPromise := promise.Catch(func(err error) any {
			panic("intentional panic in Catch callback")
		})

		// The next promise should be rejected due to panic
		_, err := nextPromise.Await()
		if err == nil {
			t.Error("Expected error due to panic, but got none")
		}
		if err.Error() != "panic occurred in error callback" {
			t.Errorf("Expected panic error, but got: %v", err)
		}
	})

	t.Run("Panic in Finally callback", func(t *testing.T) {
		promise := New(func(resolve func(string), reject func(error)) {
			resolve("success")
		})

		// Add a Finally callback that will panic
		nextPromise := promise.Finally(func() {
			panic("intentional panic in Finally callback")
		})

		// The next promise should be rejected due to panic
		_, err := nextPromise.Await()
		if err == nil {
			t.Error("Expected error due to panic, but got none")
		}
		if err.Error() != "panic occurred in finally callback" {
			t.Errorf("Expected panic error, but got: %v", err)
		}
	})

	t.Run("Error panic in callback", func(t *testing.T) {
		promise := New(func(resolve func(string), reject func(error)) {
			resolve("success")
		})

		// Add a Then callback that will panic with an error
		nextPromise := promise.Then(func(value string) any {
			panic(errors.New("custom error panic"))
		}, nil)

		// The next promise should be rejected with the custom error
		_, err := nextPromise.Await()
		if err == nil {
			t.Error("Expected error due to panic, but got none")
		}
		if err.Error() != "custom error panic" {
			t.Errorf("Expected custom error panic, but got: %v", err)
		}
	})
}
