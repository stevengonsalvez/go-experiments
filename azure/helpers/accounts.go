package azure

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/resources/mgmt/subscriptions"
	mgmtstorage "github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/storage/mgmt/storage"
	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

// getAccountClient retrieves a blob storage account client in an automated way
// based on the local Azure CLI session or SP credentials in environment vars.
//
// This function will pick the first subscription available to the user, and
// the first account with the requested name within the subscription.
func getAccountClient(
	ctx context.Context,
	accountName string,
) (storage.BlobStorageClient, error) {
	nilClient := storage.BlobStorageClient{}

	// Initialize authorizer based either on CLI or environment variables.
	authorizer, err := auth.NewAuthorizerFromCLIWithResource(
		azure.PublicCloud.ResourceManagerEndpoint,
	)
	if err != nil { // Try environment
		cfg := &auth.ClientCredentialsConfig{
			ClientID:     os.Getenv("AZURE_CLIENT_ID"),
			ClientSecret: os.Getenv("AZURE_CLIENT_SECRET"),
			TenantID:     os.Getenv("AZURE_TENANT_ID"),
			Resource:     azure.PublicCloud.ResourceManagerEndpoint,
			AADEndpoint:  azure.PublicCloud.ActiveDirectoryEndpoint,
		}
		authorizer, err = cfg.Authorizer()
		if err != nil {
			return nilClient, fmt.Errorf(
				"auth initialization failed, %v", err)
		}
	}

	// Get subscription ID.
	subID, err := inferSubscriptionIDFromUser(ctx, authorizer)
	if err != nil {
		return nilClient, fmt.Errorf(
			"could not infer subscription id for logged in user, %v", err)
	}

	// Find the resource group name of the account
	rg, err := findAccountRG(ctx, authorizer, subID, accountName)
	if err != nil {
		return nilClient, fmt.Errorf("could not find storage account, %v", err)
	}

	// Get account key
	accountKey, err := getAccountKey(ctx, authorizer, subID, rg, accountName)
	if err != nil {
		return nilClient, err
	}

	// Initialize and return client
	client, err := storage.NewBasicClient(accountName, accountKey)
	if err != nil {
		return nilClient, fmt.Errorf("client, %w", err)
	}
	return client.GetBlobService(), nil
}

// getAccountKey lists account keys and picks the first one from the list.
func getAccountKey(
	ctx context.Context,
	authorizer autorest.Authorizer,
	subscriptionID string,
	resourceGroupName string,
	accountName string,
) (string, error) {
	accountsClient := mgmtstorage.NewAccountsClient(subscriptionID)
	accountsClient.Authorizer = authorizer
	listKeysResult, err := accountsClient.ListKeys(ctx,
		resourceGroupName, accountName)
	if err != nil {
		return "", err
	}
	if len(*listKeysResult.Keys) == 0 {
		return "", errors.New("failed to list keys in storage acc")
	}
	key := *(*listKeysResult.Keys)[0].Value
	return key, nil
}

// findAccountRG lists storage accounts available within the subscription and
// returns the resourge group name of the first matching storage account, or an
// error if the storage account could not be found.
func findAccountRG(
	ctx context.Context,
	authorizer autorest.Authorizer,
	subscriptionID string,
	accountName string,
) (string, error) {
	accountsClient := mgmtstorage.NewAccountsClient(subscriptionID)
	accountsClient.Authorizer = authorizer
	result, err := accountsClient.List(ctx)
	if err != nil {
		return "", fmt.Errorf("list blob accounts err, %w", err)
	}
	for _, val := range *result.Value {
		if *val.Name == accountName {
			idparts := strings.Split(*val.ID, "/")
			if len(idparts) < 4 {
				return "", errors.New("invalid account id")
			}
			return idparts[4], nil
		}
	}
	return "", errors.New("not found")
}

// inferSubscriptionIDFromUser lists subscriptions available to the user that is
// embedded in the authorizer, and returns the first available subscription.
func inferSubscriptionIDFromUser(
	ctx context.Context,
	authorizer autorest.Authorizer,
) (string, error) {
	subscriptionClient := subscriptions.NewClient()
	subscriptionClient.Authorizer = authorizer
	res, err := subscriptionClient.List(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list subscriptions, %v", err)
	}
	subs := res.Values()
	if len(subs) < 1 {
		return "", errors.New("logged in user does not have access to any subscriptions")
	}
	for _, sub := range subs {
		return *sub.SubscriptionID, nil
	}
	return "", errors.New("subscription id not found")
}
