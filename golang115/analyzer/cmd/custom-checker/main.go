package main

import (
	"github.com/cybozu/neco-containers/golang115/analyzer/pkg/eventuallycheck"
	"github.com/cybozu/neco-containers/golang115/analyzer/pkg/restrictpkg"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(
		restrictpkg.RestrictPackageAnalyzer,
		eventuallycheck.Analyzer,
	)
}
