package ksuite

import (
	"fmt"
	"github.com/stretchr/testify/suite"
	"testing"
)

type KsuiteTestSuite struct {
	KubeSuite
}

func (k *KsuiteTestSuite) Test() {
	fmt.Println("Asd")
}

func TestKsuiteTestSuite(t *testing.T) {
	suite.Run(t, new(KsuiteTestSuite))
}
