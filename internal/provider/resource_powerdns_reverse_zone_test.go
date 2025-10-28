package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccReverseZoneResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccReverseZoneResourceConfig("172.16.0.0/16", "Master", []string{"ns1.example.com.", "ns2.example.com."}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("powerdns_reverse_zone.test", "cidr", "172.16.0.0/16"),
					resource.TestCheckResourceAttr("powerdns_reverse_zone.test", "kind", "Master"),
					resource.TestCheckResourceAttr("powerdns_reverse_zone.test", "name", "16.172.in-addr.arpa."),
					resource.TestCheckResourceAttr("powerdns_reverse_zone.test", "nameservers.#", "2"),
					resource.TestCheckResourceAttr("powerdns_reverse_zone.test", "nameservers.0", "ns1.example.com."),
					resource.TestCheckResourceAttr("powerdns_reverse_zone.test", "nameservers.1", "ns2.example.com."),
					resource.TestCheckResourceAttrSet("powerdns_reverse_zone.test", "id"),
				),
			},
			// Update and Read testing
			{
				Config: testAccReverseZoneResourceConfig("172.16.0.0/16", "Master", []string{"ns3.example.com.", "ns4.example.com."}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("powerdns_reverse_zone.test", "nameservers.#", "2"),
					resource.TestCheckResourceAttr("powerdns_reverse_zone.test", "nameservers.0", "ns3.example.com."),
					resource.TestCheckResourceAttr("powerdns_reverse_zone.test", "nameservers.1", "ns4.example.com."),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccReverseZoneResource_IPv6(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccReverseZoneResourceConfig("2001:db8::/32", "Master", []string{"ns1.example.com.", "ns2.example.com."}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("powerdns_reverse_zone.test", "cidr", "2001:db8::/32"),
					resource.TestCheckResourceAttr("powerdns_reverse_zone.test", "kind", "Master"),
					resource.TestCheckResourceAttr("powerdns_reverse_zone.test", "name", "8.b.d.0.1.0.0.2.ip6.arpa."),
					resource.TestCheckResourceAttr("powerdns_reverse_zone.test", "nameservers.#", "2"),
				),
			},
		},
	})
}

func testAccReverseZoneResourceConfig(cidr, kind string, nameservers []string) string {
	nameserversStr := ""
	for _, ns := range nameservers {
		nameserversStr += fmt.Sprintf(`"%s",`, ns)
	}
	if len(nameserversStr) > 0 {
		nameserversStr = nameserversStr[:len(nameserversStr)-1] // Remove trailing comma
	}

	return fmt.Sprintf(`
resource "powerdns_reverse_zone" "test" {
  cidr        = %[1]q
  kind        = %[2]q
  nameservers = [%[3]s]
}
`, cidr, kind, nameserversStr)
}