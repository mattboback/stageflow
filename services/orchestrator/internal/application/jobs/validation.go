package jobs

import "fmt"

type validatablePayload interface {
	Validate() error
}

func validateInboundPayload(event string, payload validatablePayload) error {
	if err := payload.Validate(); err != nil {
		return fmt.Errorf("%s payload invalid: %w", event, err)
	}

	return nil
}
