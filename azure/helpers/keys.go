package azure

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/2019-03-01/resources/mgmt/subscriptions"
	"github.com/Azure/azure-sdk-for-go/services/web/mgmt/2019-08-01/web"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

func getMasterKey(ctx context.Context, appName string) (string, error) {
	authorizer, err := auth.NewAuthorizerFromCLI()
	if err != nil {
		authorizer, err = auth.NewAuthorizerFromEnvironment()
		if err != nil {
			return "", fmt.Errorf(
				"auth initialization failed, %v", err)
		}
	}
	subscriptionID, err := findSubscriptionID(ctx, authorizer)
	if err != nil {
		return "", fmt.Errorf(
			"failed to infer subscription ID based on logged in user, %v", err)
	}
	site, err := findSite(ctx, authorizer, subscriptionID, appName)
	if err != nil {
		return "", fmt.Errorf("could not find site with name %v, %v", appName, err)
	}

	appsClient := web.NewAppsClient(subscriptionID)
	appsClient.Authorizer = authorizer
	keys, err := appsClient.ListHostKeys(ctx, *site.ResourceGroup, *site.Name)
	if err != nil {
		return "", fmt.Errorf(
			"failed to list host keys for app with name %v, %v", appName, err)
	}
	return *keys.MasterKey, nil
}

func findSubscriptionID(
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

func findSite(
	ctx context.Context,
	authorizer autorest.Authorizer,
	subscriptionID string,
	name string,
) (*web.Site, error) {
	appsClient := web.NewAppsClient(subscriptionID)
	appsClient.Authorizer = authorizer

	// Find app service that we are looking for
	apps, err := appsClient.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list web apps, %v", err)
	}
	for {
		for _, site := range apps.Values() {
			if *site.Name == name {
				return &site, nil
			}
		}
		if !apps.NotDone() {
			break
		}
		if err := apps.NextWithContext(ctx); err != nil {
			return nil, fmt.Errorf("failed fetch next page of web apps, %v", err)
		}
	}
	return nil, errors.New("site not found")
}
