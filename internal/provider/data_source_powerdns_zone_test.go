package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccZoneDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccZoneDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.powerdns_zone.test", "name", "example.com."),
					resource.TestCheckResourceAttr("data.powerdns_zone.test", "kind", "Master"),
					resource.TestCheckResourceAttrSet("data.powerdns_zone.test", "account"),
					resource.TestCheckResourceAttrSet("data.powerdns_zone.test", "soa_edit_api"),
					resource.TestCheckResourceAttrSet("data.powerdns_zone.test", "id"),
				),
			},
		},
	})
}

const testAccZoneDataSourceConfig = `
data "powerdns_zone" "test" {
  name = "example.com."
}
`