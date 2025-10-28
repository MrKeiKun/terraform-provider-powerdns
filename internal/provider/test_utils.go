package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testAccPreCheck(t *testing.T) {
	// Add any pre-check logic here if needed
	// For example, check if required environment variables are set
}

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"powerdns": func() (tfprotov6.ProviderServer, error) {
		return providerserver.NewProtocol6(New("test")())(), nil
	},
}

var testAccProvider terraform.ResourceState
