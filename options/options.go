// Package options provides a reusable parameter object for passing around global command-line options
// which need to be plumbed through different commands.
package options

// Options is a reusable parameter object used to pass global command-line options through different
// commands.
type Options struct {
	// SkipRewriting controls whether BUILD files are rewritten with linter-style updates when updates
	// are made.
	SkipRewriting bool `long:"skip_rewriting" description:"When generating build files, skip linter-style rewrites"`
}

// TestOptions provides sane default options for testing.
var TestOptions = Options{
	SkipRewriting: true,
}
