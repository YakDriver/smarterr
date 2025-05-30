package internal

// As a error handling library, smarterr should produce a minimum of
// its own errors, failing silently when possible. However, in order
// to have assurances, we need to validate the configuration and
// proactively catch errors prior to runtime. This allows CI/CD
// validation instead of runtime errors.

// Validate will eventually enable a command-line validation of the
// smarterr configuration. It should do at least these things:
// 1. Ensure that a token.parameter matches a parameter in the config
// 2. Ensure that token.stack_matches match stack_match blocks in the config
