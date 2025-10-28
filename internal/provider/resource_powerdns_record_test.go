package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRecordResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccRecordResourceConfig("example.com.", "test.example.com.", "A", 300, []string{"192.168.1.1"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("powerdns_record.test", "zone", "example.com."),
					resource.TestCheckResourceAttr("powerdns_record.test", "name", "test.example.com."),
					resource.TestCheckResourceAttr("powerdns_record.test", "type", "A"),
					resource.TestCheckResourceAttr("powerdns_record.test", "ttl", "300"),
					resource.TestCheckResourceAttr("powerdns_record.test", "records.#", "1"),
					resource.TestCheckResourceAttr("powerdns_record.test", "records.0", "192.168.1.1"),
					resource.TestCheckResourceAttrSet("powerdns_record.test", "id"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccRecordResource_CNAME(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccRecordResourceConfig("example.com.", "alias.example.com.", "CNAME", 3600, []string{"target.example.com."}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("powerdns_record.test", "zone", "example.com."),
					resource.TestCheckResourceAttr("powerdns_record.test", "name", "alias.example.com."),
					resource.TestCheckResourceAttr("powerdns_record.test", "type", "CNAME"),
					resource.TestCheckResourceAttr("powerdns_record.test", "ttl", "3600"),
					resource.TestCheckResourceAttr("powerdns_record.test", "records.#", "1"),
					resource.TestCheckResourceAttr("powerdns_record.test", "records.0", "target.example.com."),
				),
			},
		},
	})
}

func TestAccRecordResource_MultipleValues(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccRecordResourceConfig("example.com.", "multi.example.com.", "A", 300, []string{"192.168.1.1", "192.168.1.2"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("powerdns_record.test", "zone", "example.com."),
					resource.TestCheckResourceAttr("powerdns_record.test", "name", "multi.example.com."),
					resource.TestCheckResourceAttr("powerdns_record.test", "type", "A"),
					resource.TestCheckResourceAttr("powerdns_record.test", "ttl", "300"),
					resource.TestCheckResourceAttr("powerdns_record.test", "records.#", "2"),
				),
			},
		},
	})
}

func testAccRecordResourceConfig(zone, name, recordType string, ttl int64, records []string) string {
	recordsStr := ""
	for _, record := range records {
		recordsStr += fmt.Sprintf(`"%s",`, record)
	}
	if len(recordsStr) > 0 {
		recordsStr = recordsStr[:len(recordsStr)-1] // Remove trailing comma
	}

	return fmt.Sprintf(`
resource "powerdns_record" "test" {
  zone    = %[1]q
  name    = %[2]q
  type    = %[3]q
  ttl     = %[4]d
  records = [%[5]s]
}
`, zone, name, recordType, ttl, recordsStr)
}
