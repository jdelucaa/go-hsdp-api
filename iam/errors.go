package iam

import "fmt"

var (
	errNotFound                    = fmt.Errorf("entity not found")
	errMissingManagingOrganization = fmt.Errorf("missing managing organization")
	errMissingName                 = fmt.Errorf("missing name value")
	errMissingDescription          = fmt.Errorf("missing description value")
	errMalformedInputValue         = fmt.Errorf("malformed input value")
	errMissingOrganization         = fmt.Errorf("missing organization")
	errMissingGlobalReference      = fmt.Errorf("missing global reference")
	errNotImplementedByHSDP        = fmt.Errorf("method not implemented by HSDP")
)
