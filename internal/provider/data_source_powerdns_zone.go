package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var _ datasource.DataSource = &ZoneDataSource{}

// ZoneDataSource defines the data source implementation.
type ZoneDataSource struct {
	client *Client
}

// ZoneDataSourceModel describes the data source data model.
type ZoneDataSourceModel struct {
	Name        types.String `tfsdk:"name"`
	Kind        types.String `tfsdk:"kind"`
	Account     types.String `tfsdk:"account"`
	Nameservers types.Set    `tfsdk:"nameservers"`
	Masters     types.Set    `tfsdk:"masters"`
	SoaEditAPI  types.String `tfsdk:"soa_edit_api"`
	Records     types.List   `tfsdk:"records"`
	ID          types.String `tfsdk:"id"`
}

func (d *ZoneDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone"
}

func (d *ZoneDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the zone to retrieve",
				Required:            true,
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "The kind of zone (Master, Slave, etc.)",
				Computed:            true,
			},
			"account": schema.StringAttribute{
				MarkdownDescription: "The account associated with the zone",
				Computed:            true,
			},
			"nameservers": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of nameservers for this zone",
				Computed:            true,
			},
			"masters": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of master servers for this zone (Slave zones only)",
				Computed:            true,
			},
			"soa_edit_api": schema.StringAttribute{
				MarkdownDescription: "SOA edit API setting",
				Computed:            true,
			},
			"records": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of all records in the zone",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Zone identifier",
			},
		},
	}
}

func (d *ZoneDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", "Expected *Client")
		return
	}
	d.client = client
}

func (d *ZoneDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ZoneDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneName := data.Name.ValueString()
	ctx = tflog.SetField(ctx, "zone_name", zoneName)
	tflog.Info(ctx, "Reading zone data source")

	// Get the zone information
	zone, err := d.client.GetZone(ctx, zoneName)
	if err != nil {
		resp.Diagnostics.AddError("Couldn't fetch zone", err.Error())
		return
	}

	// Check if zone exists
	if zone.Name == "" {
		resp.Diagnostics.AddError("Zone not found", fmt.Sprintf("zone %s not found", zoneName))
		return
	}

	ctx = tflog.SetField(ctx, "kind", zone.Kind)
	tflog.Info(ctx, "Found zone")

	// Set zone information
	data.ID = types.StringValue(zone.Name)
	data.Name = types.StringValue(zone.Name)
	data.Kind = types.StringValue(zone.Kind)
	data.Account = types.StringValue(zone.Account)
	data.SoaEditAPI = types.StringValue(zone.SoaEditAPI)

	// Set nameservers for non-Slave zones
	if !strings.EqualFold(zone.Kind, "Slave") {
		nameservers, err := d.client.ListRecordsInRRSet(ctx, zoneName, zoneName, "NS")
		if err != nil {
			resp.Diagnostics.AddError("Couldn't fetch nameservers", err.Error())
			return
		}

		var zoneNameservers []types.String
		for _, ns := range nameservers {
			zoneNameservers = append(zoneNameservers, types.StringValue(ns.Content))
		}

		data.Nameservers, _ = types.SetValueFrom(ctx, types.StringType, zoneNameservers)
	}

	// Set masters for Slave zones
	if strings.EqualFold(zone.Kind, "Slave") {
		var masters []types.String
		for _, master := range zone.Masters {
			masters = append(masters, types.StringValue(master))
		}
		data.Masters, _ = types.SetValueFrom(ctx, types.StringType, masters)
	}

	// Get all records in the zone
	allRecords, err := d.client.ListRecords(ctx, zoneName)
	if err != nil {
		resp.Diagnostics.AddError("Couldn't fetch records", err.Error())
		return
	}

	// Convert records to simple string format to avoid nested object complexity
	var recordStrings []string
	for _, r := range allRecords {
		recordStr := fmt.Sprintf("%s %d %s %s", r.Name, r.TTL, r.Type, r.Content)
		recordStrings = append(recordStrings, recordStr)
	}

	// For now, just store records as a list of strings
	// In a production system, we'd want proper nested object support
	data.Records, _ = types.ListValueFrom(ctx, types.StringType, recordStrings)

	tflog.Info(ctx, "Successfully retrieved zone records", map[string]interface{}{
		"record_count": len(recordStrings),
	})
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func NewZoneDataSource() datasource.DataSource {
	return &ZoneDataSource{}
}
