package sui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// FaucetRequest represents the body of a faucet request
type FaucetRequest struct {
	FixedAmountRequest struct {
		Recipient string `json:"recipient"`
	} `json:"FixedAmountRequest"`
}

// RequestFaucet sends a POST request to the faucet endpoint to request gas for the given address.
func RequestFaucet(faucetURL, address string) error {
	reqBody := FaucetRequest{}
	reqBody.FixedAmountRequest.Recipient = address

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal faucet request: %w", err)
	}

	resp, err := http.Post(faucetURL+"/gas", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send faucet request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("faucet request failed with status: %s", resp.Status)
	}

	return nil
}
