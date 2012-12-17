package dag

import (
	"errors"
)

// stub implementation of BuildRule for use in unit tests
type stubrule struct {
	// takes name of first target -- used for recording order in which
	// targets are built
	callback func(string)

	targets  []Node
	fail     bool
	executed bool
}

func makestubrule(callback func(string), target ...Node) *stubrule {
	return &stubrule{
		callback: callback,
		targets:  target,
	}
}

func (self *stubrule) Execute() ([]Node, []error) {
	self.callback(self.targets[0].String())
	errs := []error{}
	if self.fail {
		errs = append(errs, errors.New("action failed"))
	}
	return self.targets, errs
}

func (self *stubrule) ActionString() string {
	return "build " + self.targets[0].String()
}
