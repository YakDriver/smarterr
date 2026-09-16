package internal

//go:fix inline
func stringPtr(s string) *string {
	return new(s)
}
