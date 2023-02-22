package azure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

func findResourceGroup(
	ctx context.Context,
	accountName string,
	subscriptionID string,
	token string,
) (string, error) {
	var jsonResp struct {
		Value []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"value"`
	}
	url := fmt.Sprintf("https://management.azure.com/subscriptions/%v/providers/Microsoft.Storage/storageAccounts?api-version=2021-04-01",
		subscriptionID,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	if err := json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		return "", err
	}
	for _, val := range jsonResp.Value {
		if val.Name == accountName {
			parts := strings.Split(val.ID, "/")
			if len(parts) < 4 {
				return "", fmt.Errorf("failed to parse account resource name")
			}
			return parts[3], nil
		}
	}
	return "", errors.New("coult not find storage account")
}

func findSubscriptionID(
	ctx context.Context,
	token string,
) (string, error) {
	var jsonResp struct {
		Value []struct {
			SubscriptionID string `json:"subscriptionId"`
		} `json:"value"`
	}
	url := "https://management.azure.com/subscriptions?api-version=2020-01-01"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	if err := json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		return "", err
	}
	if len(jsonResp.Value) == 0 {
		return "", errors.New("no subscriptions available for service principal")
	}
	return jsonResp.Value[0].SubscriptionID, nil
}

func maybeGetTokenFromCLI() (string, error) {
	token, err := cli.GetTokenFromCLI(azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return "", err
	}
	adalToken, err := token.ToADALToken()
	if err != nil {
		return "", err
	}
	return adalToken.AccessToken, nil
}

func maybeGetTokenFromEnv() (string, error) {
	for _, envName := range []string{
		"AZURE_CLIENT_ID",
		"AZURE_CLIENT_SECRET",
		"AZURE_TENANT_ID",
	} {
		if os.Getenv(envName) == "" {
			return "", fmt.Errorf("azure token from env failed, %v is unset", envName)
		}
	}
	cfg := &auth.ClientCredentialsConfig{
		ClientID:     os.Getenv("AZURE_CLIENT_ID"),
		ClientSecret: os.Getenv("AZURE_CLIENT_SECRET"),
		TenantID:     os.Getenv("AZURE_TENANT_ID"),
		Resource:     azure.PublicCloud.ResourceManagerEndpoint,
		AADEndpoint:  azure.PublicCloud.ActiveDirectoryEndpoint,
	}
	token, err := cfg.ServicePrincipalToken()
	if err != nil {
		return "", err
	}
	if err := token.EnsureFresh(); err != nil {
		return "", err
	}
	return token.OAuthToken(), nil
}
