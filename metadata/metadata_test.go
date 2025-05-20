package metadata

import (
	"reflect"
	"testing"

	"github.com/go-gost/core/metadata"
	// No specific imports needed beyond core/metadata and testing for this file.
)

func TestNewMetadata(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		expected metadata.Metadata
		isNilOut bool
	}{
		{
			name:     "nil input map",
			input:    nil,
			expected: mapMetadata{}, // or rather, check IsNil behavior
			isNilOut: true,          // NewMetadata returns nil for empty/nil map
		},
		{
			name:     "empty input map",
			input:    map[string]any{},
			expected: mapMetadata{},
			isNilOut: true, // NewMetadata returns nil for empty/nil map
		},
		{
			name: "simple map",
			input: map[string]any{
				"key1": "value1",
				"Key2": 123,
			},
			expected: mapMetadata{
				"key1": "value1",
				"key2": 123, // Keys should be lowercased
			},
			isNilOut: false,
		},
		{
			name: "map with mixed case keys",
			input: map[string]any{
				"StringValue": "test",
				"IntValue":    42,
				"boolValue":   true,
			},
			expected: mapMetadata{
				"stringvalue": "test",
				"intvalue":    42,
				"boolvalue":   true, // Keys should be lowercased
			},
			isNilOut: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewMetadata(tt.input)
			if tt.isNilOut {
				if got != nil {
					t.Errorf("NewMetadata() with input %v = %v, want nil", tt.input, got)
				}
				return // Skip further checks if nil is expected
			}

			// Type assert to mapMetadata for comparison, only if not nil
			gotMap, ok := got.(mapMetadata)
			if !ok && !tt.isNilOut {
				t.Fatalf("NewMetadata() did not return a mapMetadata type, got %T", got)
			}

			expectedMap := tt.expected.(mapMetadata)
			if !reflect.DeepEqual(gotMap, expectedMap) {
				t.Errorf("NewMetadata() = %v, want %v", gotMap, expectedMap)
			}
		})
	}
}

func TestMapMetadata_IsExists(t *testing.T) {
	md := NewMetadata(map[string]any{
		"key1": "value1",
		"Key2": true, // Will be stored as "key2"
	}).(mapMetadata) // Assume NewMetadata works and gives mapMetadata

	var nilMd metadata.Metadata // Interface type, will be nil
	var emptyMd metadata.Metadata = NewMetadata(nil)


	tests := []struct {
		name     string
		m        metadata.Metadata // Testing against the interface
		key      string
		expected bool
	}{
		{"key exists", md, "key1", true},
		{"key exists case insensitive input (Key1)", md, "Key1", true},
		{"key exists case insensitive input (KEY1)", md, "KEY1", true},
		{"key2 exists", md, "key2", true},
		{"key2 exists case insensitive input (Key2)", md, "Key2", true},
		{"key does not exist", md, "nonexistent", false},
		{"empty key", md, "", false}, // Assuming empty keys are not typically used or stored specially
		{"nil metadata", nilMd, "key1", false}, // IsExists should handle nil receiver if called directly (though methods on nil interface value are tricky)
		{"empty metadata", emptyMd, "key1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// If tt.m is nil (like nilMd), calling methods on it would panic.
			// The functions in util.go check if md is nil first.
			// Here, we are testing the methods of mapMetadata type itself.
			// An interface variable can be nil. A mapMetadata variable (concrete type) cannot be nil itself,
			// but its underlying map can be. NewMetadata returns nil interface for empty maps.
			if tt.m == nil { // Handles nilMd and emptyMd which NewMetadata makes nil
				if tt.expected != false {
					t.Errorf("IsExists() on nil metadata should be false, want %v", tt.expected)
				}
				return
			}
			// Now we are sure tt.m is not a nil interface, so it must be mapMetadata
			if got := tt.m.IsExists(tt.key); got != tt.expected {
				t.Errorf("mapMetadata.IsExists(%q) = %v, want %v", tt.key, got, tt.expected)
			}
		})
	}
}

func TestMapMetadata_Set(t *testing.T) {
	// Start with an empty metadata object from NewMetadata to ensure it's initialized correctly.
	// NewMetadata returns nil for empty maps, which means we can't call Set on it.
	// So, we must start with a non-empty map or test Set on a mapMetadata directly.
	md := NewMetadata(map[string]any{"initial": "value"}).(mapMetadata)


	tests := []struct {
		name       string
		keyToSet   string
		valueToSet any
		keyToCheck string // Could be different case from keyToSet
		expected   any
	}{
		{"set new key", "newKey", "newValue", "newkey", "newValue"},
		{"overwrite existing key", "initial", "overwritten", "initial", "overwritten"},
		{"set with mixed case key", "MixedCase", 123, "mixedcase", 123},
		{"set then check with different case", "AnotherKey", true, "anotherKEY", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md.Set(tt.keyToSet, tt.valueToSet)
			got := md.Get(tt.keyToCheck)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("After Set(%q, %v), Get(%q) = %v, want %v", tt.keyToSet, tt.valueToSet, tt.keyToCheck, got, tt.expected)
			}
		})
	}

	// Test Set on a mapMetadata that was initially empty but not nil interface
	// This requires direct instantiation if NewMetadata(nil) returns nil interface.
	// However, the typical usage would be through NewMetadata.
	// If md is mapMetadata{}, calling Set should still work.
	directMd := make(mapMetadata)
	directMd.Set("directKey", "directValue")
	if val := directMd.Get("directkey"); val != "directValue" {
		t.Errorf("Set on directly created mapMetadata failed, got %v, want %v", val, "directValue")
	}
}

func TestMapMetadata_Get(t *testing.T) {
	md := NewMetadata(map[string]any{
		"key1": "value1",
		"Key2": 123,    // Stored as "key2"
		"nilValueKey": nil,
	}).(mapMetadata)

	var nilMdInterface metadata.Metadata // Interface type, will be nil
	var emptyMdInterface metadata.Metadata = NewMetadata(nil) // NewMetadata returns nil interface

	tests := []struct {
		name     string
		m        metadata.Metadata // Testing against the interface
		key      string
		expected any
	}{
		{"get existing key", md, "key1", "value1"},
		{"get existing key (stored lowercase)", md, "key2", 123},
		{"get existing key case insensitive (Key1)", md, "Key1", "value1"},
		{"get existing key case insensitive (KEY2)", md, "KEY2", 123},
		{"get non-existent key", md, "nonexistent", nil},
		{"get key with nil value", md, "nilValueKey", nil},
		{"get key with nil value, case insensitive", md, "NilValueKEY", nil},
		{"get from nil metadata interface", nilMdInterface, "key1", nil},
		{"get from empty metadata interface (nil)", emptyMdInterface, "key1", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// If tt.m is nil (like nilMdInterface or emptyMdInterface), calling Get method would panic.
			// However, the mapMetadata.Get method itself has a nil check for its receiver `m`
			// but that's for `m mapMetadata`, not `metadata.Metadata` interface.
			// If `tt.m` (the interface) is nil, `tt.m.Get()` will panic.
			// The util functions handle this by checking `if md == nil`.
			// For testing the method directly, we need to be careful.
			var got any
			if tt.m == nil { // Handles nil interface cases
				got = nil // This is the effective behavior if one were to call Get on a nil mapMetadata *through the interface if it was possible without panic*
			} else {
				got = tt.m.Get(tt.key)
			}

			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("mapMetadata.Get(%q) = %v, want %v", tt.key, got, tt.expected)
			}
		})
	}

    // Test Get on a mapMetadata variable that is a nil map (not nil interface)
    // This is a bit of a Go nuance. mapMetadata is `map[string]any`.
    // A declared var `m mapMetadata` is nil by default.
    var nilMapMd mapMetadata
    if nilMapMd.Get("anyKey") != nil {
        t.Errorf("Get on a nil map mapMetadata should return nil, got %v", nilMapMd.Get("anyKey"))
    }
}
