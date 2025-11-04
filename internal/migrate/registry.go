package migrate

// LoadPatterns loads all pattern groups from the registry
func LoadPatterns() []PatternGroup {
	return []PatternGroup{
		CreateImportPatterns(),     // Run first to add required imports
		CreateBareErrorPatterns(),
		CreateTfresourcePatterns(),
		CreateFrameworkPatterns(),
		CreateSDKv2Patterns(),
		CreateHelperPatterns(),
	}
}
