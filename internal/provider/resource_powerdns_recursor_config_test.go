package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRecursorConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccRecursorConfigResourceConfig("test-config", "test-value"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("powerdns_recursor_config.test", "name", "test-config"),
					resource.TestCheckResourceAttr("powerdns_recursor_config.test", "value", "test-value"),
					resource.TestCheckResourceAttrSet("powerdns_recursor_config.test", "id"),
				),
			},
			// Update and Read testing
			{
				Config: testAccRecursorConfigResourceConfig("test-config", "updated-value"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("powerdns_recursor_config.test", "name", "test-config"),
					resource.TestCheckResourceAttr("powerdns_recursor_config.test", "value", "updated-value"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccRecursorConfigResourceConfig(name, value string) string {
	return `
resource "powerdns_recursor_config" "test" {
  name  = "` + name + `"
  value = "` + value + `"
}
`
}
