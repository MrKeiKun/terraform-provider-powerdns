package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRecursorForwardZoneResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccRecursorForwardZoneResourceConfig("example.com.", []string{"8.8.8.8", "8.8.4.4"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("powerdns_recursor_forward_zone.test", "zone", "example.com."),
					resource.TestCheckResourceAttr("powerdns_recursor_forward_zone.test", "servers.#", "2"),
					resource.TestCheckResourceAttr("powerdns_recursor_forward_zone.test", "servers.0", "8.8.8.8"),
					resource.TestCheckResourceAttr("powerdns_recursor_forward_zone.test", "servers.1", "8.8.4.4"),
					resource.TestCheckResourceAttrSet("powerdns_recursor_forward_zone.test", "id"),
				),
			},
			// Update and Read testing
			{
				Config: testAccRecursorForwardZoneResourceConfig("example.com.", []string{"1.1.1.1", "9.9.9.9"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("powerdns_recursor_forward_zone.test", "zone", "example.com."),
					resource.TestCheckResourceAttr("powerdns_recursor_forward_zone.test", "servers.#", "2"),
					resource.TestCheckResourceAttr("powerdns_recursor_forward_zone.test", "servers.0", "1.1.1.1"),
					resource.TestCheckResourceAttr("powerdns_recursor_forward_zone.test", "servers.1", "9.9.9.9"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccRecursorForwardZoneResourceConfig(zone string, servers []string) string {
	serversStr := ""
	for _, server := range servers {
		serversStr += fmt.Sprintf(`"%s",`, server)
	}
	if len(serversStr) > 0 {
		serversStr = serversStr[:len(serversStr)-1] // Remove trailing comma
	}

	return fmt.Sprintf(`
resource "powerdns_recursor_forward_zone" "test" {
  zone    = %[1]q
  servers = [%[2]s]
}
`, zone, serversStr)
}