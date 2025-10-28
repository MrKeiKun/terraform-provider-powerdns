package provider

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var _ resource.Resource = &ZoneResource{}

// ZoneResource defines the resource implementation.
type ZoneResource struct {
	client *Client
}

// ZoneResourceModel describes the resource data model.
type ZoneResourceModel struct {
	Name        types.String `tfsdk:"name"`
	Kind        types.String `tfsdk:"kind"`
	Account     types.String `tfsdk:"account"`
	Nameservers types.Set    `tfsdk:"nameservers"`
	Masters     types.Set    `tfsdk:"masters"`
	SoaEditAPI  types.String `tfsdk:"soa_edit_api"`
	ID          types.String `tfsdk:"id"`
}

func (r *ZoneResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zone"
}

func (r *ZoneResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the zone",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "The kind of the zone",
				Required:            true,
			},
			"account": schema.StringAttribute{
				MarkdownDescription: "The account owning the zone",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"nameservers": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of zone nameservers",
				Optional:            true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
			"masters": schema.SetAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of IP addresses configured as a master for this zone",
				Optional:            true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
			"soa_edit_api": schema.StringAttribute{
				MarkdownDescription: "SOA edit API setting",
				Optional:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Zone identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ZoneResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *Client")
		return
	}
	r.client = client
}

func (r *ZoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ZoneResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate masters for Slave zones
	if data.Kind.ValueString() == "Slave" && data.Masters.IsNull() {
		resp.Diagnostics.AddError("Missing required attribute", "masters attribute is required for Slave zones")
		return
	}

	// Convert nameservers
	var nameservers []string
	if !data.Nameservers.IsNull() {
		for _, ns := range data.Nameservers.Elements() {
			if str, ok := ns.(types.String); ok {
				nameservers = append(nameservers, str.ValueString())
			}
		}
	}

	// Convert and validate masters
	var masters []string
	if !data.Masters.IsNull() {
		for _, master := range data.Masters.Elements() {
			if str, ok := master.(types.String); ok {
				masterStr := str.ValueString()
				splitIPPort := strings.Split(masterStr, ":")
				if len(splitIPPort) > 2 {
					resp.Diagnostics.AddError("Invalid master format", "More than one colon in <ip>:<port> string")
					return
				}
				if len(splitIPPort) == 2 {
					port, err := strconv.Atoi(splitIPPort[1])
					if err != nil {
						resp.Diagnostics.AddError("Invalid port", "Error converting port value in masters attribute")
						return
					}
					if port < 1 || port > 65535 {
						resp.Diagnostics.AddError("Invalid port", "Port value must be between 1 and 65535")
						return
					}
				}
				masterIP := splitIPPort[0]
				if net.ParseIP(masterIP) == nil {
					resp.Diagnostics.AddError("Invalid IP", "Values in masters list must be valid IPs")
					return
				}
				masters = append(masters, masterStr)
			}
		}
	}

	zoneInfo := ZoneInfo{
		Name:        data.Name.ValueString(),
		Kind:        data.Kind.ValueString(),
		Account:     data.Account.ValueString(),
		Nameservers: nameservers,
		SoaEditAPI:  data.SoaEditAPI.ValueString(),
	}

	if len(masters) > 0 {
		if strings.EqualFold(zoneInfo.Kind, "Slave") {
			zoneInfo.Masters = masters
		} else {
			resp.Diagnostics.AddError("Invalid configuration", "masters attribute is supported only for Slave kind")
			return
		}
	}

	tflog.SetField(ctx, "zone_name", zoneInfo.Name)
	tflog.SetField(ctx, "zone_kind", zoneInfo.Kind)
	tflog.Debug(ctx, "Creating PowerDNS Zone")

	createdZoneInfo, err := r.client.CreateZone(ctx, zoneInfo)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create zone", err.Error())
		return
	}

	data.ID = types.StringValue(createdZoneInfo.ID)
	tflog.Info(ctx, "Created PowerDNS Zone", map[string]any{"id": createdZoneInfo.ID})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ZoneResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.SetField(ctx, "zone_id", data.ID.ValueString())
	tflog.Debug(ctx, "Reading PowerDNS Zone")

	zoneInfo, err := r.client.GetZone(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read zone", fmt.Errorf("couldn't fetch PowerDNS Zone: %w", err).Error())
		return
	}

	if zoneInfo.Name == "" {
		tflog.Warn(ctx, "Zone not found; removing from state")
		resp.State.RemoveResource(ctx)
		return
	}

	data.Name = types.StringValue(zoneInfo.Name)
	data.Kind = types.StringValue(zoneInfo.Kind)
	data.Account = types.StringValue(zoneInfo.Account)
	data.SoaEditAPI = types.StringValue(zoneInfo.SoaEditAPI)

	// Only manage NS records for non-Slave zones
	if !strings.EqualFold(zoneInfo.Kind, "Slave") {
		nameservers, err := r.client.ListRecordsInRRSet(ctx, zoneInfo.Name, zoneInfo.Name, "NS")
		if err != nil {
			resp.Diagnostics.AddError("Failed to read nameservers", fmt.Errorf("couldn't fetch zone %s nameservers from PowerDNS: %w", zoneInfo.Name, err).Error())
			return
		}

		var zoneNameservers []types.String
		for _, nameserver := range nameservers {
			zoneNameservers = append(zoneNameservers, types.StringValue(nameserver.Content))
		}

		data.Nameservers, _ = types.SetValueFrom(ctx, types.StringType, zoneNameservers)
	}

	if strings.EqualFold(zoneInfo.Kind, "Slave") {
		var masters []types.String
		for _, master := range zoneInfo.Masters {
			masters = append(masters, types.StringValue(master))
		}
		data.Masters, _ = types.SetValueFrom(ctx, types.StringType, masters)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ZoneResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.SetField(ctx, "zone_id", data.ID.ValueString())
	tflog.Debug(ctx, "Updating PowerDNS Zone")

	zoneInfo := ZoneInfoUpd{
		Name:       data.Name.ValueString(),
		Kind:       data.Kind.ValueString(),
		Account:    data.Account.ValueString(),
		SoaEditAPI: data.SoaEditAPI.ValueString(),
	}

	if err := r.client.UpdateZone(ctx, data.ID.ValueString(), zoneInfo); err != nil {
		resp.Diagnostics.AddError("Failed to update zone", fmt.Errorf("error updating PowerDNS Zone: %w", err).Error())
		return
	}

	// Read the updated state
	updatedZoneInfo, err := r.client.GetZone(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read updated zone", fmt.Errorf("couldn't fetch PowerDNS Zone: %w", err).Error())
		return
	}

	data.Name = types.StringValue(updatedZoneInfo.Name)
	data.Kind = types.StringValue(updatedZoneInfo.Kind)
	data.Account = types.StringValue(updatedZoneInfo.Account)
	data.SoaEditAPI = types.StringValue(updatedZoneInfo.SoaEditAPI)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ZoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ZoneResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.SetField(ctx, "zone_id", data.ID.ValueString())
	tflog.Debug(ctx, "Deleting PowerDNS Zone")

	if err := r.client.DeleteZone(ctx, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete zone", fmt.Errorf("error deleting PowerDNS Zone: %w", err).Error())
		return
	}

	tflog.Info(ctx, "Deleted PowerDNS Zone")
}

func (r *ZoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func NewZoneResource() resource.Resource {
	return &ZoneResource{}
}
