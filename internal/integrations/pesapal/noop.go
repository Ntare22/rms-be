package pesapal

import (
	"context"
	"fmt"
)

// NoopClient is used when PesaPal credentials are not configured.
type NoopClient struct{}

func (NoopClient) RequestToken(context.Context) (string, error) {
	return "", fmt.Errorf("pesapal is not configured")
}

func (NoopClient) SubmitOrder(context.Context, SubmitOrderRequest) (*SubmitOrderResponse, error) {
	return nil, fmt.Errorf("pesapal is not configured")
}

func (NoopClient) RegisterIPN(context.Context, string, string) (*RegisterIPNResponse, error) {
	return nil, fmt.Errorf("pesapal is not configured")
}

func (NoopClient) GetIPNList(context.Context) ([]IPNListItem, error) {
	return nil, fmt.Errorf("pesapal is not configured")
}

func (NoopClient) GetTransactionStatus(context.Context, string) (*TransactionStatusResponse, error) {
	return nil, fmt.Errorf("pesapal is not configured")
}
