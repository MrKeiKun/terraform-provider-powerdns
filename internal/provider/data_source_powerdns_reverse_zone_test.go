package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourcePDNSReverseZone_basic(t *testing.T) {
	cidr := "192.168.1.0/24"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePDNSReverseZoneConfig(cidr),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.powerdns_reverse_zone.test", "cidr", cidr),
					resource.TestCheckResourceAttr("data.powerdns_reverse_zone.test", "name", "1.168.192.in-addr.arpa."),
					resource.TestCheckResourceAttrSet("data.powerdns_reverse_zone.test", "kind"),
					resource.TestCheckResourceAttrSet("data.powerdns_reverse_zone.test", "nameservers.#"),
					resource.TestCheckResourceAttrSet("data.powerdns_reverse_zone.test", "id"),
				),
			},
		},
	})
}

func TestAccDataSourcePDNSReverseZone_notFound(t *testing.T) {
	cidr := "10.0.0.0/8"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourcePDNSReverseZoneNotFoundConfig(cidr),
				ExpectError: regexp.MustCompile("Couldn't fetch zone"),
			},
		},
	})
}

func TestAccDataSourcePDNSReverseZone_invalidCIDR(t *testing.T) {
	cidr := "invalid-cidr"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourcePDNSReverseZoneConfig(cidr),
				ExpectError: regexp.MustCompile(fmt.Sprintf("invalid CIDR address: %s", cidr)),
			},
		},
	})
}

func testAccDataSourcePDNSReverseZoneConfig(cidr string) string {
	return fmt.Sprintf(`
provider "powerdns" {
  server_url         = "http://localhost:8081"
  recursor_server_url = "http://localhost:8082"
  api_key            = "secret"
}

resource "powerdns_reverse_zone" "test_reverse_zone" {
  cidr        = %[1]q
  kind        = "Master"
  nameservers = ["ns1.test.example.com.", "ns2.test.example.com."]
}

data "powerdns_reverse_zone" "test" {
  cidr = %[1]q
  depends_on = [powerdns_reverse_zone.test_reverse_zone]
}
`, cidr)
}

func testAccDataSourcePDNSReverseZoneNotFoundConfig(cidr string) string {
	return fmt.Sprintf(`
provider "powerdns" {
  server_url         = "http://localhost:8081"
  recursor_server_url = "http://localhost:8082"
  api_key            = "secret"
}

data "powerdns_reverse_zone" "test" {
  cidr = %[1]q
}
`, cidr)
}
