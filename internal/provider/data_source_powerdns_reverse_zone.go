package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var _ datasource.DataSource = &ReverseZoneDataSource{}

// ReverseZoneDataSource defines the data source implementation.
type ReverseZoneDataSource struct {
	client *Client
}

// ReverseZoneDataSourceModel describes the data source data model.
type ReverseZoneDataSourceModel struct {
	Cidr        types.String `tfsdk:"cidr"`
	Kind        types.String `tfsdk:"kind"`
	Nameservers types.List   `tfsdk:"nameservers"`
	Name        types.String `tfsdk:"name"`
	ID          types.String `tfsdk:"id"`
}

func (d *ReverseZoneDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_reverse_zone"
}

func (d *ReverseZoneDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"cidr": schema.StringAttribute{
				MarkdownDescription: "The CIDR block for the reverse zone (e.g., '172.16.0.0/16')",
				Required:            true,
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "The kind of zone (Master or Slave)",
				Computed:            true,
			},
			"nameservers": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of nameservers for this zone",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The computed zone name (e.g., '16.172.in-addr.arpa.')",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Zone identifier",
			},
		},
	}
}

func (d *ReverseZoneDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ReverseZoneDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ReverseZoneDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cidr := data.Cidr.ValueString()
	ctx = tflog.SetField(ctx, "cidr", cidr)
	tflog.Info(ctx, "Reading reverse zone data source")

	zoneName, err := GetReverseZoneName(cidr)
	if err != nil {
		resp.Diagnostics.AddError("Failed to determine zone name", err.Error())
		return
	}
	ctx = tflog.SetField(ctx, "zone_name", zoneName)
	tflog.Debug(ctx, "Computed reverse zone name from CIDR")

	zone, err := d.client.GetZone(ctx, zoneName)
	if err != nil {
		resp.Diagnostics.AddError("Couldn't fetch zone", err.Error())
		return
	}

	// Check if zone exists by checking if the name is empty
	if zone.Name == "" {
		resp.Diagnostics.AddError("Reverse zone not found", fmt.Sprintf("reverse zone for CIDR %s not found", cidr))
		return
	}

	tflog.Info(ctx, "Found reverse zone", map[string]interface{}{
		"name": zone.Name,
		"kind": zone.Kind,
	})

	data.ID = types.StringValue(zone.Name)
	data.Name = types.StringValue(zone.Name)
	data.Kind = types.StringValue(zone.Kind)

	// Read nameservers from NS records
	nameservers, err := d.client.ListRecordsInRRSet(ctx, zoneName, zoneName, "NS")
	if err != nil {
		resp.Diagnostics.AddError("Couldn't fetch nameservers", err.Error())
		return
	}

	var zoneNameservers []types.String
	for _, ns := range nameservers {
		zoneNameservers = append(zoneNameservers, types.StringValue(ns.Content))
	}

	data.Nameservers, _ = types.ListValueFrom(ctx, types.StringType, zoneNameservers)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func NewReverseZoneDataSource() datasource.DataSource {
	return &ReverseZoneDataSource{}
}
