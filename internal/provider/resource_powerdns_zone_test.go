package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccZoneResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccZoneResourceConfig("example.com.", "Master"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("powerdns_zone.test", "name", "example.com."),
					resource.TestCheckResourceAttr("powerdns_zone.test", "kind", "Master"),
					resource.TestCheckResourceAttrSet("powerdns_zone.test", "id"),
				),
			},
			// Update and Read testing
			{
				Config: testAccZoneResourceConfig("example.com.", "Master"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("powerdns_zone.test", "name", "example.com."),
					resource.TestCheckResourceAttr("powerdns_zone.test", "kind", "Master"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccZoneResource_Slave(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccZoneResourceSlaveConfig("slave.example.com.", "Slave", []string{"192.168.1.1:53"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("powerdns_zone.test", "name", "slave.example.com."),
					resource.TestCheckResourceAttr("powerdns_zone.test", "kind", "Slave"),
					resource.TestCheckResourceAttr("powerdns_zone.test", "masters.#", "1"),
					resource.TestCheckResourceAttr("powerdns_zone.test", "masters.0", "192.168.1.1:53"),
				),
			},
		},
	})
}

func testAccZoneResourceConfig(name, kind string) string {
	return fmt.Sprintf(`
resource "powerdns_zone" "test" {
  name = %[1]q
  kind = %[2]q
}
`, name, kind)
}

func testAccZoneResourceSlaveConfig(name, kind string, masters []string) string {
	mastersStr := ""
	for _, master := range masters {
		mastersStr += fmt.Sprintf(`"%s",`, master)
	}
	if len(mastersStr) > 0 {
		mastersStr = mastersStr[:len(mastersStr)-1] // Remove trailing comma
	}

	return fmt.Sprintf(`
resource "powerdns_zone" "test" {
  name    = %[1]q
  kind    = %[2]q
  masters = [%[3]s]
}
`, name, kind, mastersStr)
}
