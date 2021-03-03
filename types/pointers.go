package types

// StringPtr returns a pointer to a string
func StringPtr(x string) *string {
	return &x
}

// IntPtr returns a pointer to an int
func IntPtr(x int) *int {
	return &x
}

// Int64Ptr returns a pointer to an int64
func Int64Ptr(x int64) *int64 {
	return &x
}

// Float32Ptr returns a pointer to a float32
func Float32Ptr(x float32) *float32 {
	return &x
}

// BoolPtr returns a pointer to a bool
func BoolPtr(x bool) *bool {
	return &x
}
