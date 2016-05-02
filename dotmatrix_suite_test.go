package dotmatrix_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestDotmatrix(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Dotmatrix Suite")
}
