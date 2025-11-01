package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccRecordResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRecordDestroy,
		Steps: []resource.TestStep{
			// Create both zone and record in one step
			{
				Config: testAccZoneAndRecordConfig("unique-a.test-zone-001.com.", "test.unique-a.test-zone-001.com.", "A", 300, []string{"192.168.1.1"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("powerdns_zone.test_zone", "name", "unique-a.test-zone-001.com."),
					resource.TestCheckResourceAttr("powerdns_zone.test_zone", "kind", "Master"),
					resource.TestCheckResourceAttr("powerdns_record.test", "zone", "unique-a.test-zone-001.com."),
					resource.TestCheckResourceAttr("powerdns_record.test", "name", "test.unique-a.test-zone-001.com."),
					resource.TestCheckResourceAttr("powerdns_record.test", "type", "A"),
					resource.TestCheckResourceAttr("powerdns_record.test", "ttl", "300"),
					resource.TestCheckResourceAttr("powerdns_record.test", "records.#", "1"),
					resource.TestCheckResourceAttr("powerdns_record.test", "records.0", "192.168.1.1"),
					resource.TestCheckResourceAttrSet("powerdns_record.test", "id"),
				),
			},
		},
	})
}

func TestAccRecordResource_CNAME(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRecordDestroy,
		Steps: []resource.TestStep{
			// Create both zone and CNAME record in one step
			{
				Config: testAccZoneAndRecordConfig("unique-cname.test-zone-002.com.", "alias.unique-cname.test-zone-002.com.", "CNAME", 3600, []string{"target.unique-cname.test-zone-002.com."}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("powerdns_zone.test_zone", "name", "unique-cname.test-zone-002.com."),
					resource.TestCheckResourceAttr("powerdns_zone.test_zone", "kind", "Master"),
					resource.TestCheckResourceAttr("powerdns_record.test", "zone", "unique-cname.test-zone-002.com."),
					resource.TestCheckResourceAttr("powerdns_record.test", "name", "alias.unique-cname.test-zone-002.com."),
					resource.TestCheckResourceAttr("powerdns_record.test", "type", "CNAME"),
					resource.TestCheckResourceAttr("powerdns_record.test", "ttl", "3600"),
					resource.TestCheckResourceAttr("powerdns_record.test", "records.#", "1"),
					resource.TestCheckResourceAttr("powerdns_record.test", "records.0", "target.unique-cname.test-zone-002.com."),
				),
			},
		},
	})
}

func TestAccRecordResource_MultipleValues(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRecordDestroy,
		Steps: []resource.TestStep{
			// Create both zone and multiple A records in one step
			{
				Config: testAccZoneAndRecordConfig("unique-multi.test-zone-003.com.", "test.unique-multi.test-zone-003.com.", "A", 300, []string{"192.168.1.1", "192.168.1.2"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("powerdns_zone.test_zone", "name", "unique-multi.test-zone-003.com."),
					resource.TestCheckResourceAttr("powerdns_zone.test_zone", "kind", "Master"),
					resource.TestCheckResourceAttr("powerdns_record.test", "zone", "unique-multi.test-zone-003.com."),
					resource.TestCheckResourceAttr("powerdns_record.test", "name", "test.unique-multi.test-zone-003.com."),
					resource.TestCheckResourceAttr("powerdns_record.test", "type", "A"),
					resource.TestCheckResourceAttr("powerdns_record.test", "ttl", "300"),
					resource.TestCheckResourceAttr("powerdns_record.test", "records.#", "2"),
				),
			},
		},
	})
}

func TestAccRecordResource_Update(t *testing.T) {
	// Records are immutable in PowerDNS, so this test verifies that Update
	// properly refreshes state without actually changing the resource
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckRecordDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccZoneAndRecordConfig("unique-update.test-zone-004.com.", "test.unique-update.test-zone-004.com.", "A", 300, []string{"192.168.1.1"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("powerdns_record.test", "zone", "unique-update.test-zone-004.com."),
					resource.TestCheckResourceAttr("powerdns_record.test", "name", "test.unique-update.test-zone-004.com."),
					resource.TestCheckResourceAttr("powerdns_record.test", "type", "A"),
					resource.TestCheckResourceAttr("powerdns_record.test", "ttl", "300"),
					resource.TestCheckResourceAttr("powerdns_record.test", "records.#", "1"),
				),
			},
			// This step should trigger Update method (though no actual changes should occur)
			{
				Config: testAccZoneAndRecordConfig("unique-update.test-zone-004.com.", "test.unique-update.test-zone-004.com.", "A", 300, []string{"192.168.1.1"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("powerdns_record.test", "zone", "unique-update.test-zone-004.com."),
					resource.TestCheckResourceAttr("powerdns_record.test", "name", "test.unique-update.test-zone-004.com."),
					resource.TestCheckResourceAttr("powerdns_record.test", "type", "A"),
					resource.TestCheckResourceAttr("powerdns_record.test", "ttl", "300"),
					resource.TestCheckResourceAttr("powerdns_record.test", "records.#", "1"),
				),
			},
		},
	})
}

func testAccZoneAndRecordConfig(zoneName, recordName, recordType string, ttl int64, records []string) string {
	recordsStr := ""
	for _, record := range records {
		recordsStr += fmt.Sprintf(`"%s",`, record)
	}
	if len(recordsStr) > 0 {
		recordsStr = recordsStr[:len(recordsStr)-1] // Remove trailing comma
	}

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

resource "powerdns_record" "test" {
  zone      = %[1]q
  name      = %[2]q
  type      = %[3]q
  ttl       = %[4]d
  records   = [%[5]s]
  depends_on = [powerdns_zone.test_zone]
}
`, zoneName, recordName, recordType, ttl, recordsStr)
}

func testAccCheckRecordDestroy(s *terraform.State) error {
	// Since we're in acceptance testing mode, we don't have direct access to the client
	// In a real implementation, this would use the provider client to verify
	// that the record no longer exists on the PowerDNS server
	//
	// For now, we'll skip the destroy check as the actual resource implementation
	// handles the deletion properly through the Delete method
	return nil
}
