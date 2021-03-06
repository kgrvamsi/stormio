package envsuite

// Provides an EnvSuite type which makes sure this test suite gets an isolated
// environment settings. Settings will be saved on start and then cleared, and
// reset on tear down.

import (
	. "launchpad.net/gocheck"
	"os"
	"strings"
)

type EnvSuite struct {
	environ []string
}

func (s *EnvSuite) SetUpSuite(c *C) {
	s.environ = os.Environ()
}

func (s *EnvSuite) SetUpTest(c *C) {
	os.Clearenv()
}

func (s *EnvSuite) TearDownTest(c *C) {
	for _, envstring := range s.environ {
		kv := strings.SplitN(envstring, "=", 2)
		os.Setenv(kv[0], kv[1])
	}
}

func (s *EnvSuite) TearDownSuite(c *C) {
}
