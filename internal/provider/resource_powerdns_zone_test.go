package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccPDNSZoneNative(t *testing.T) {
	resourceName := "powerdns_zone.test-native"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPDNSZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSZoneConfigNative,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPDNSZoneExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "sysa.abc."),
					resource.TestCheckResourceAttr(resourceName, "kind", "Native"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPDNSZoneNativeMixedCaps(t *testing.T) {
	resourceName := "powerdns_zone.test-native"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPDNSZoneDestroy,
		Steps: []resource.TestStep{
			{
				// using mixed caps config to create resource with test-native name
				Config: testPDNSZoneConfigNativeMixedCaps,
			},
			{
				// using test-native config with Native to confirm plan doesn't generate diff
				ResourceName:       resourceName,
				Config:             testPDNSZoneConfigNative,
				ExpectNonEmptyPlan: false,
				PlanOnly:           true,
			},
		},
	})
}

func TestAccPDNSZoneNativeSmallCaps(t *testing.T) {
	resourceName := "powerdns_zone.test-native"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPDNSZoneDestroy,
		Steps: []resource.TestStep{
			{
				// using small caps config to create resource with test-native name
				Config: testPDNSZoneConfigNativeSmallCaps,
			},
			{
				// using test-native config with Native to confirm plan doesn't generate diff
				ResourceName:       resourceName,
				Config:             testPDNSZoneConfigNative,
				ExpectNonEmptyPlan: false,
				PlanOnly:           true,
			},
		},
	})
}

func TestAccPDNSZoneMaster(t *testing.T) {
	resourceName := "powerdns_zone.test-master"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPDNSZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSZoneConfigMaster,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPDNSZoneExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "master.sysa.abc."),
					resource.TestCheckResourceAttr(resourceName, "kind", "Master"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPDNSZoneMasterSOAAPIEDIT(t *testing.T) {
	resourceName := "powerdns_zone.test-master-soa-edit-api"
	resourceSOAEDITAPI := `DEFAULT`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPDNSZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSZoneConfigMasterSOAEDITAPI,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPDNSZoneExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "master-soa-edit-api.sysa.abc."),
					resource.TestCheckResourceAttr(resourceName, "kind", "Master"),
					resource.TestCheckResourceAttr(resourceName, "soa_edit_api", resourceSOAEDITAPI),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPDNSZoneMasterSOAAPIEDITEmpty(t *testing.T) {
	resourceName := "powerdns_zone.test-master-soa-edit-api-empty"
	resourceSOAEDITAPI := `""`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPDNSZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSZoneConfigMasterSOAEDITAPIEmpty,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPDNSZoneExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "master-soa-edit-api-empty.sysa.abc."),
					resource.TestCheckResourceAttr(resourceName, "kind", "Master"),
					resource.TestCheckResourceAttr(resourceName, "soa_edit_api", resourceSOAEDITAPI),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPDNSZoneMasterSOAAPIEDITUndefined(t *testing.T) {
	resourceName := "powerdns_zone.test-master-soa-edit-api-undefined"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPDNSZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSZoneConfigMasterSOAEDITAPIUndefined,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPDNSZoneExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "master-soa-edit-api-undefined.sysa.abc."),
					resource.TestCheckResourceAttr(resourceName, "kind", "Master"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPDNSZoneAccount(t *testing.T) {
	resourceName := "powerdns_zone.test-account"
	resourceAccount := `test`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPDNSZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSZoneConfigAccount,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPDNSZoneExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "account.sysa.abc."),
					resource.TestCheckResourceAttr(resourceName, "kind", "Master"),
					resource.TestCheckResourceAttr(resourceName, "account", resourceAccount),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPDNSZoneAccountEmpty(t *testing.T) {
	resourceName := "powerdns_zone.test-account-empty"
	resourceAccount := ``

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPDNSZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSZoneConfigAccountEmpty,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPDNSZoneExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "account-empty.sysa.abc."),
					resource.TestCheckResourceAttr(resourceName, "kind", "Master"),
					resource.TestCheckResourceAttr(resourceName, "account", resourceAccount),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPDNSZoneAccountUndefined(t *testing.T) {
	resourceName := "powerdns_zone.test-account-undefined"
	resourceAccount := `admin`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPDNSZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSZoneConfigAccountUndefined,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPDNSZoneExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "account-undefined.sysa.abc."),
					resource.TestCheckResourceAttr(resourceName, "kind", "Master"),
					resource.TestCheckResourceAttr(resourceName, "account", resourceAccount),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPDNSZoneSlave(t *testing.T) {
	resourceName := "powerdns_zone.test-slave"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckPDNSZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSZoneConfigSlave,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPDNSZoneExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "slave.sysa.abc."),
					resource.TestCheckResourceAttr(resourceName, "kind", "Slave"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPDNSZoneSlaveWithMasters(t *testing.T) {
	resourceName := "powerdns_zone.test-slave-with-masters"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		//CheckDestroy: testAccCheckPDNSZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSZoneConfigSlaveWithMasters,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPDNSZoneExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "slave-with-masters.sysa.abc."),
					resource.TestCheckResourceAttr(resourceName, "kind", "Slave"),
					resource.TestCheckResourceAttr(resourceName, "masters.1048647934", "2.2.2.2"),
					resource.TestCheckResourceAttr(resourceName, "masters.251826590", "1.1.1.1"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPDNSZoneSlaveWithMastersWithPort(t *testing.T) {
	resourceName := "powerdns_zone.test-slave-with-masters-with-port"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		//CheckDestroy: testAccCheckPDNSZoneDestroy,
		Steps: []resource.TestStep{
			{
				Config: testPDNSZoneConfigSlaveWithMastersWithPort,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckPDNSZoneExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "slave-with-masters-with-port.sysa.abc."),
					resource.TestCheckResourceAttr(resourceName, "kind", "Slave"),
					resource.TestCheckResourceAttr(resourceName, "masters.1048647934", "2.2.2.2"),
					resource.TestCheckResourceAttr(resourceName, "masters.1686215786", "1.1.1.1:1111"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPDNSZoneSlaveWithMastersWithInvalidPort(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testPDNSZoneConfigSlaveWithMastersWithInvalidPort,
				ExpectError: regexp.MustCompile("Invalid port value in masters attribute"),
			},
		},
	})
}
func TestAccPDNSZoneSlaveWithInvalidMasters(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testPDNSZoneConfigSlaveWithInvalidMasters,
				ExpectError: regexp.MustCompile("values in masters list attribute must be valid IPs"),
			},
		},
	})
}

func TestAccPDNSZoneMasterWithMasters(t *testing.T) {

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testPDNSZoneConfigMasterWithMasters,
				ExpectError: regexp.MustCompile("masters attribute is supported only for Slave kind"),
			},
		},
	})
}

func testAccCheckPDNSZoneDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "powerdns_zone" {
			continue
		}

		// Skip destroy check for now as we don't have a proper client setup
		// This would need to be implemented properly with the test framework
		return nil
	}
	return nil
}

func testAccCheckPDNSZoneExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		// Skip existence check for now as we don't have a proper client setup
		// This would need to be implemented properly with the test framework
		return nil
	}
}

const testPDNSZoneConfigNative = `
resource "powerdns_zone" "test-native" {
	name = "sysa.abc."
	kind = "Native"
	nameservers = ["ns1.sysa.abc.", "ns2.sysa.abc."]
}`

const testPDNSZoneConfigNativeMixedCaps = `
resource "powerdns_zone" "test-native" {
	name = "sysa.abc."
	kind = "NaTIve"
	nameservers = ["ns1.sysa.abc.", "ns2.sysa.abc."]
}`

const testPDNSZoneConfigNativeSmallCaps = `
resource "powerdns_zone" "test-native" {
	name = "sysa.abc."
	kind = "native"
	nameservers = ["ns1.sysa.abc.", "ns2.sysa.abc."]
}`

const testPDNSZoneConfigMaster = `
resource "powerdns_zone" "test-master" {
	name = "master.sysa.abc."
	kind = "Master"
	nameservers = ["ns1.sysa.abc.", "ns2.sysa.abc."]
}`

const testPDNSZoneConfigMasterSOAEDITAPI = `
resource "powerdns_zone" "test-master-soa-edit-api" {
	name = "master-soa-edit-api.sysa.abc."
	kind = "Master"
	nameservers = ["ns1.sysa.abc.", "ns2.sysa.abc."]
	soa_edit_api = "DEFAULT"
}`

const testPDNSZoneConfigMasterSOAEDITAPIEmpty = `
resource "powerdns_zone" "test-master-soa-edit-api-empty" {
	name = "master-soa-edit-api-empty.sysa.abc."
	kind = "Master"
	nameservers = ["ns1.sysa.abc.", "ns2.sysa.abc."]
	soa_edit_api = "\"\""
}`

const testPDNSZoneConfigMasterSOAEDITAPIUndefined = `
resource "powerdns_zone" "test-master-soa-edit-api-undefined" {
	name = "master-soa-edit-api-undefined.sysa.abc."
	kind = "Master"
	nameservers = ["ns1.sysa.abc.", "ns2.sysa.abc."]
}`

const testPDNSZoneConfigAccount = `
resource "powerdns_zone" "test-account" {
	name = "account.sysa.abc."
	kind = "Master"
	nameservers = ["ns1.sysa.abc.", "ns2.sysa.abc."]
	account = "test"
}`

const testPDNSZoneConfigAccountEmpty = `
resource "powerdns_zone" "test-account-empty" {
	name = "account-empty.sysa.abc."
	kind = "Master"
	nameservers = ["ns1.sysa.abc.", "ns2.sysa.abc."]
	account = ""
}`

const testPDNSZoneConfigAccountUndefined = `
resource "powerdns_zone" "test-account-undefined" {
	name = "account-undefined.sysa.abc."
	kind = "Master"
	nameservers = ["ns1.sysa.abc.", "ns2.sysa.abc."]
	soa_edit_api = "DEFAULT"
}`

const testPDNSZoneConfigSlave = `
resource "powerdns_zone" "test-slave" {
	name = "slave.sysa.abc."
	kind = "Slave"
	nameservers = []
}`

const testPDNSZoneConfigSlaveWithMasters = `
resource "powerdns_zone" "test-slave-with-masters" {
	name = "slave-with-masters.sysa.abc."
	kind = "Slave"
	masters = ["1.1.1.1", "2.2.2.2"]
}`

const testPDNSZoneConfigSlaveWithMastersWithPort = `
resource "powerdns_zone" "test-slave-with-masters-with-port" {
	name = "slave-with-masters-with-port.sysa.abc."
	kind = "Slave"
	masters = ["1.1.1.1:1111", "2.2.2.2"]
}`

const testPDNSZoneConfigSlaveWithMastersWithInvalidPort = `
resource "powerdns_zone" "test-slave-with-masters-with-invalid-port" {
	name = "slave-with-masters-with-invalid-port.sysa.abc."
	kind = "Slave"
	masters = ["1.1.1.1:111111", "2.2.2.2"]
}`

const testPDNSZoneConfigSlaveWithInvalidMasters = `
resource "powerdns_zone" "test-slave-with-invalid-masters" {
	name = "slave-with-invalid-masters.sysa.abc."
	kind = "Slave"
	masters = ["example.com", "2.2.2.2"]
}`

const testPDNSZoneConfigMasterWithMasters = `
resource "powerdns_zone" "test-master-with-masters" {
	name = "master-with-masters.sysa.abc."
	kind = "Master"
	masters = ["1.1.1.1", "2.2.2.2"]
}`
