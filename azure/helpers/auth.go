package azure

import (
	"fmt"
	"os"

	"github.com/Azure/azure-sdk-for-go/services/web/mgmt/2019-08-01/web"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

func Setup() {
	// Load Azure credentials from environment variables
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	client = web.NewAppsClient(subscriptionID)
	authorizer, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		fmt.Println("Failed to get authorizer from environment, \n check to see if you have set the correct environment variables" +
			"AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, AZURE_TENANT_ID, AZURE_SUBSCRIPTION_ID")
		fmt.Println(err)
	}
	client.Authorizer = authorizer
}
