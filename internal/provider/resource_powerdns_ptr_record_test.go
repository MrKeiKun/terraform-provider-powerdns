package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPTRRecordResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPTRRecordResourceConfig("192.168.1.10", "host.example.com.", 300, "1.168.192.in-addr.arpa."),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("powerdns_ptr_record.test", "ip_address", "192.168.1.10"),
					resource.TestCheckResourceAttr("powerdns_ptr_record.test", "hostname", "host.example.com."),
					resource.TestCheckResourceAttr("powerdns_ptr_record.test", "ttl", "300"),
					resource.TestCheckResourceAttr("powerdns_ptr_record.test", "reverse_zone", "1.168.192.in-addr.arpa."),
					resource.TestCheckResourceAttrSet("powerdns_ptr_record.test", "id"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccPTRRecordResource_IPv6(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccPTRRecordResourceConfig("2001:db8::1", "ipv6host.example.com.", 3600, "0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa."),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("powerdns_ptr_record.test", "ip_address", "2001:db8::1"),
					resource.TestCheckResourceAttr("powerdns_ptr_record.test", "hostname", "ipv6host.example.com."),
					resource.TestCheckResourceAttr("powerdns_ptr_record.test", "ttl", "3600"),
					resource.TestCheckResourceAttr("powerdns_ptr_record.test", "reverse_zone", "0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa."),
				),
			},
		},
	})
}

func testAccPTRRecordResourceConfig(ipAddress, hostname string, ttl int64, reverseZone string) string {
	return fmt.Sprintf(`
resource "powerdns_ptr_record" "test" {
  ip_address   = %[1]q
  hostname     = %[2]q
  ttl          = %[3]d
  reverse_zone = %[4]q
}
`, ipAddress, hostname, ttl, reverseZone)
}