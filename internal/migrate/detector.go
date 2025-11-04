package migrate

import "regexp"

// MigrationDetector handles detection of code that needs migration
type MigrationDetector struct{}

// NewMigrationDetector creates a new migration detector
func NewMigrationDetector() *MigrationDetector {
	return &MigrationDetector{}
}

// MigrationPatterns defines the patterns that indicate code needs migration
var MigrationPatterns = []string{
	`response\.Diagnostics\.Append`,
	`response\.Diagnostics\.AddError`,
	`sdkdiag\.AppendFromErr`,
	`sdkdiag\.AppendErrorf`,
	`create\.AppendDiagError`,
	`create\.AddError`,
	`create\.ProblemStandardMessage`,
	`return.*fmt\.Errorf`,
	`fmt\.Errorf.*(?i)unexpected format`,
	`return append\(diags,`,
	`tfresource\.NotFound`,
	`return nil, "", err`,
	`(?m)return nil, err$`, // Use multiline mode
	`return nil, &retry\.NotFoundError`,
	`return nil, tfresource\.NewEmptyResultError`,
	`return tfresource\.AssertSingleValueResult`,
}

// NeedsMigration checks if the given content contains patterns that need migration
func (md *MigrationDetector) NeedsMigration(content string) bool {
	for _, pattern := range MigrationPatterns {
		matched, _ := regexp.MatchString(pattern, content)
		if matched {
			return true
		}
	}
	return false
}

// NeedsMigration is a convenience function for checking if content needs migration
func NeedsMigration(content string) bool {
	detector := NewMigrationDetector()
	return detector.NeedsMigration(content)
}
