package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccReverseZoneDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccReverseZoneDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.powerdns_reverse_zone.test", "cidr", "172.16.0.0/16"),
					resource.TestCheckResourceAttr("data.powerdns_reverse_zone.test", "name", "16.172.in-addr.arpa."),
					resource.TestCheckResourceAttr("data.powerdns_reverse_zone.test", "kind", "Master"),
					resource.TestCheckResourceAttrSet("data.powerdns_reverse_zone.test", "id"),
				),
			},
		},
	})
}

const testAccReverseZoneDataSourceConfig = `
data "powerdns_reverse_zone" "test" {
  cidr = "172.16.0.0/16"
}
`