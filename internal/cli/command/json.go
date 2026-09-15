package command

import (
	"encoding/json"
	"fmt"
	"io"
)

func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("write JSON: %w", err)
	}
	return nil
}
