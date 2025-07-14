package main

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/manifoldco/promptui"
	"golang.org/x/oauth2/clientcredentials"
)

func main() {
	// Prompt for Azure and AWS parameters
	prompt := promptui.Prompt{Label: "Azure Client ID"}
	azureClientID, _ := prompt.Run()
	prompt = promptui.Prompt{Label: "Azure Client Secret", Mask: '*'}
	azureClientSecret, _ := prompt.Run()
	prompt = promptui.Prompt{Label: "Azure Tenant ID"}
	azureTenantID, _ := prompt.Run()
	prompt = promptui.Prompt{Label: "Azure Scope (e.g. api://AWS_SSO_APP_ID/.default)"}
	azureScope, _ := prompt.Run()
	prompt = promptui.Prompt{Label: "AWS SSO Region"}
	awsRegion, _ := prompt.Run()

	// Get Azure token
	conf := &clientcredentials.Config{
		ClientID:     azureClientID,
		ClientSecret: azureClientSecret,
		TokenURL:     fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", azureTenantID),
		Scopes:       []string{azureScope},
	}
	token, err := conf.Token(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Azure token error: %v\n", err)
		return
	}
	azureAccessToken := token.AccessToken

	// AWS SSO client
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(awsRegion))
	if err != nil {
		fmt.Fprintf(os.Stderr, "AWS config error: %v\n", err)
		return
	}
	client := sso.NewFromConfig(cfg)

	// List accounts
	accounts, err := client.ListAccounts(context.TODO(), &sso.ListAccountsInput{
		AccessToken: aws.String(azureAccessToken),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListAccounts error: %v\n", err)
		return
	}
	if len(accounts.AccountList) == 0 {
		fmt.Println("No accounts found.")
		return
	}

	// Select account
	accountNames := make([]string, len(accounts.AccountList))
	for i, acct := range accounts.AccountList {
		accountNames[i] = fmt.Sprintf("%s (%s)", *acct.AccountName, *acct.AccountId)
	}
	accountPrompt := promptui.Select{
		Label: "Select AWS Account",
		Items: accountNames,
	}
	accountIdx, _, err := accountPrompt.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Account selection error: %v\n", err)
		return
	}
	selectedAccount := accounts.AccountList[accountIdx]

	// List roles for selected account
	roles, err := client.ListAccountRoles(context.TODO(), &sso.ListAccountRolesInput{
		AccessToken: aws.String(azureAccessToken),
		AccountId:   selectedAccount.AccountId,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ListAccountRoles error: %v\n", err)
		return
	}
	if len(roles.RoleList) == 0 {
		fmt.Println("No roles found for this account.")
		return
	}

	// Select role
	roleNames := make([]string, len(roles.RoleList))
	for i, role := range roles.RoleList {
		roleNames[i] = *role.RoleName
	}
	rolePrompt := promptui.Select{
		Label: "Select Role",
		Items: roleNames,
	}
	roleIdx, _, err := rolePrompt.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Role selection error: %v\n", err)
		return
	}
	selectedRole := roles.RoleList[roleIdx]

	fmt.Printf("Selected Account: %s\n", *selectedAccount.AccountId)
	fmt.Printf("Selected Role: %s\n", *selectedRole.RoleName)
}
