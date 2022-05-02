package testutil

// Int32Ptr returns the provided 32-bit integer as a pointer to a 32-bit integer
func Int32Ptr(i int32) *int32 {
	return &i
}

// Int64Ptr returns the provided 64-bit integer as a pointer to a 64-bit integer
func Int64Ptr(i int64) *int64 {
	return &i
}
