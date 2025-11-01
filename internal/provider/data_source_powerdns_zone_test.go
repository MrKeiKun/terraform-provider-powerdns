package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourcePDNSZone_basic(t *testing.T) {
	zoneName := "example.com."

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePDNSZoneConfig(zoneName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDataSourcePDNSZoneCheck("data.powerdns_zone.test", zoneName),
				),
			},
		},
	})
}

func TestAccDataSourcePDNSZone_withRecords(t *testing.T) {
	zoneName := "example.com."

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePDNSZoneConfigWithRecords(zoneName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccDataSourcePDNSZoneCheckWithRecords("data.powerdns_zone.test", zoneName),
				),
			},
		},
	})
}

func testAccDataSourcePDNSZoneConfig(zoneName string) string {
	return fmt.Sprintf(`
provider "powerdns" {
  server_url         = "http://localhost:8081"
  recursor_server_url = "http://localhost:8082"
  api_key            = "secret"
}

resource "powerdns_zone" "test_zone" {
  name        = %[1]q
  kind        = "Master"
  nameservers = ["ns1.test.example.com.", "ns2.test.example.com."]
}

data "powerdns_zone" "test" {
  name = %[1]q
  depends_on = [powerdns_zone.test_zone]
}
`, zoneName)
}

func testAccDataSourcePDNSZoneConfigWithRecords(zoneName string) string {
	return fmt.Sprintf(`
provider "powerdns" {
  server_url         = "http://localhost:8081"
  recursor_server_url = "http://localhost:8082"
  api_key            = "secret"
}

resource "powerdns_zone" "test_zone" {
  name        = %[1]q
  kind        = "Master"
  nameservers = ["ns1.test.example.com.", "ns2.test.example.com."]
}

data "powerdns_zone" "test" {
  name = %[1]q
  depends_on = [powerdns_zone.test_zone]
}

output "zone_records" {
  value = data.powerdns_zone.test.records
}

output "a_records" {
  value = data.powerdns_zone.test.records
}
`, zoneName)
}

func testAccDataSourcePDNSZoneCheck(dataSourceName, zoneName string) resource.TestCheckFunc {
	return resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr(dataSourceName, "name", zoneName),
		resource.TestCheckResourceAttrSet(dataSourceName, "kind"),
		resource.TestCheckResourceAttrSet(dataSourceName, "records.#"),
	)
}

func testAccDataSourcePDNSZoneCheckWithRecords(dataSourceName, zoneName string) resource.TestCheckFunc {
	return resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr(dataSourceName, "name", zoneName),
		resource.TestCheckResourceAttrSet(dataSourceName, "kind"),
		resource.TestCheckResourceAttrSet(dataSourceName, "records.#"),
	)
}
