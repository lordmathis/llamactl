package testutil

import "slices"

// Helper functions for pointer fields
func BoolPtr(b bool) *bool {
	return &b
}

func IntPtr(i int) *int {
	return &i
}

// Helper functions for testing command arguments

// Contains checks if a slice contains a specific item
func Contains(slice []string, item string) bool {
	return slices.Contains(slice, item)
}

// ContainsFlagWithValue checks if args contains a flag followed by a specific value
func ContainsFlagWithValue(args []string, flag, value string) bool {
	for i, arg := range args {
		if arg == flag {
			// Check if there's a next argument and it matches the expected value
			if i+1 < len(args) && args[i+1] == value {
				return true
			}
		}
	}
	return false
}
