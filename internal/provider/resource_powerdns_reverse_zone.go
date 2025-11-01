package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// CIDRValidator implements a custom validator for CIDR format.
type CIDRValidator struct{}

func (v CIDRValidator) Description(ctx context.Context) string {
	return "Validates CIDR format according to PowerDNS reverse zone requirements"
}

func (v CIDRValidator) MarkdownDescription(ctx context.Context) string {
	return "Validates CIDR format according to PowerDNS reverse zone requirements"
}

func (v CIDRValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	cidr := req.ConfigValue.ValueString()

	// Use the existing ValidateCIDR function
	_, errors := ValidateCIDR(cidr, "cidr")
	if len(errors) > 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid CIDR",
			fmt.Sprintf("Invalid CIDR format: %v", errors[0]),
		)
	}
}

// Ensure the implementation satisfies the expected interfaces.
var _ resource.Resource = &ReverseZoneResource{}

// ReverseZoneResource defines the resource implementation.
type ReverseZoneResource struct {
	client *Client
}

// ReverseZoneResourceModel describes the resource data model.
type ReverseZoneResourceModel struct {
	CIDR        types.String `tfsdk:"cidr"`
	Kind        types.String `tfsdk:"kind"`
	Nameservers types.List   `tfsdk:"nameservers"`
	Name        types.String `tfsdk:"name"`
	ID          types.String `tfsdk:"id"`
}

func (r *ReverseZoneResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_reverse_zone"
}

func (r *ReverseZoneResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"cidr": schema.StringAttribute{
				MarkdownDescription: "The CIDR block for the reverse zone",
				Required:            true,
				Validators: []validator.String{
					CIDRValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "The kind of zone (Master or Slave)",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"nameservers": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of nameservers for this zone",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The computed zone name",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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

func (r *ReverseZoneResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ReverseZoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ReverseZoneResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cidr := data.CIDR.ValueString()
	tflog.SetField(ctx, "cidr", cidr)
	tflog.Debug(ctx, "Creating reverse zone")

	zoneName, err := GetReverseZoneName(cidr)
	if err != nil {
		resp.Diagnostics.AddError("Failed to determine zone name", fmt.Errorf("failed to determine zone name: %w", err).Error())
		return
	}
	tflog.Info(ctx, "Generated reverse zone name", map[string]any{"zone": zoneName})

	// Convert nameservers
	var nameservers []string
	if !data.Nameservers.IsNull() {
		for _, ns := range data.Nameservers.Elements() {
			if str, ok := ns.(types.String); ok {
				nameservers = append(nameservers, str.ValueString())
			}
		}
	}

	zone := ZoneInfo{
		Name:        zoneName,
		Kind:        data.Kind.ValueString(),
		Nameservers: nameservers,
	}

	createdZone, err := r.client.CreateZone(ctx, zone)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create reverse zone", fmt.Errorf("failed to create reverse zone: %w", err).Error())
		return
	}

	data.ID = types.StringValue(createdZone.Name)
	data.Name = types.StringValue(createdZone.Name)
	tflog.Info(ctx, "Created reverse zone", map[string]any{"id": createdZone.Name})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ReverseZoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ReverseZoneResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneName := data.ID.ValueString()

	tflog.SetField(ctx, "zone", zoneName)
	tflog.Debug(ctx, "Reading reverse zone")

	zone, err := r.client.GetZone(ctx, zoneName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read zone", fmt.Errorf("couldn't fetch zone: %w", err).Error())
		return
	}

	// If zone doesn't exist, clear state
	if zone.Name == "" {
		tflog.Warn(ctx, "Zone not found; removing from state")
		resp.State.RemoveResource(ctx)
		return
	}

	tflog.Info(ctx, "Found reverse zone", map[string]any{"zone": zone.Name, "kind": zone.Kind})

	data.Name = types.StringValue(zone.Name)
	data.Kind = types.StringValue(zone.Kind)

	// Read nameservers from NS records
	nameservers, err := r.client.ListRecordsInRRSet(ctx, zoneName, zoneName, "NS")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read nameservers", fmt.Errorf("couldn't fetch zone %s nameservers from PowerDNS: %w", zoneName, err).Error())
		return
	}

	var zoneNameservers []types.String
	for _, ns := range nameservers {
		zoneNameservers = append(zoneNameservers, types.StringValue(ns.Content))
	}

	data.Nameservers, _ = types.ListValueFrom(ctx, types.StringType, zoneNameservers)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ReverseZoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ReverseZoneResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneName := data.ID.ValueString()

	tflog.SetField(ctx, "zone", zoneName)
	tflog.Debug(ctx, "Updating reverse zone")

	// Get current zone info
	zone, err := r.client.GetZone(ctx, zoneName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read zone", fmt.Errorf("couldn't fetch zone: %w", err).Error())
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

	// Update nameservers in zone object
	zone.Nameservers = nameservers

	// Build update request
	zoneInfo := ZoneInfoUpd{
		Name:       zoneName,
		Kind:       zone.Kind,
		Account:    zone.Account,
		SoaEditAPI: zone.SoaEditAPI,
	}

	if err := r.client.UpdateZone(ctx, zoneName, zoneInfo); err != nil {
		resp.Diagnostics.AddError("Failed to update zone", fmt.Errorf("error updating zone: %w", err).Error())
		return
	}

	// Update NS records to reflect nameserver list
	rrSet := ResourceRecordSet{
		Name:       zoneName,
		Type:       "NS",
		TTL:        3600,
		ChangeType: "REPLACE",
		Records:    make([]Record, len(nameservers)),
	}

	for i, ns := range nameservers {
		rrSet.Records[i] = Record{
			Content: ns,
			TTL:     3600,
		}
	}

	if _, err := r.client.ReplaceRecordSet(ctx, zoneName, rrSet); err != nil {
		resp.Diagnostics.AddError("Failed to update nameserver records", fmt.Errorf("error updating nameserver records: %w", err).Error())
		return
	}

	tflog.Info(ctx, "Updated reverse zone")

	// Read the updated state
	updatedZone, err := r.client.GetZone(ctx, zoneName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read updated zone", fmt.Errorf("couldn't fetch zone: %w", err).Error())
		return
	}

	data.Name = types.StringValue(updatedZone.Name)
	data.Kind = types.StringValue(updatedZone.Kind)

	// Read updated nameservers
	nameserversRecords, err := r.client.ListRecordsInRRSet(ctx, zoneName, zoneName, "NS")
	if err != nil {
		resp.Diagnostics.AddError("Failed to read nameservers", fmt.Errorf("couldn't fetch zone %s nameservers from PowerDNS: %w", zoneName, err).Error())
		return
	}

	var updatedNameservers []types.String
	for _, ns := range nameserversRecords {
		updatedNameservers = append(updatedNameservers, types.StringValue(ns.Content))
	}

	data.Nameservers, _ = types.ListValueFrom(ctx, types.StringType, updatedNameservers)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ReverseZoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ReverseZoneResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zoneName := data.ID.ValueString()

	tflog.SetField(ctx, "zone", zoneName)
	tflog.Debug(ctx, "Deleting reverse zone")

	if err := r.client.DeleteZone(ctx, zoneName); err != nil {
		resp.Diagnostics.AddError("Failed to delete zone", fmt.Errorf("error deleting zone: %w", err).Error())
		return
	}

	tflog.Info(ctx, "Deleted reverse zone")
}

func (r *ReverseZoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	zoneName := req.ID
	tflog.Info(ctx, "Importing reverse zone", map[string]any{"zone": zoneName})

	cidr, err := ParseReverseZoneName(zoneName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse reverse zone name", err.Error())
		return
	}

	zone, err := r.client.GetZone(ctx, zoneName)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get zone", fmt.Errorf("error getting zone: %w", err).Error())
		return
	}

	// Convert nameservers
	var nameservers []types.String
	for _, ns := range zone.Nameservers {
		nameservers = append(nameservers, types.StringValue(ns))
	}

	var dataModel ReverseZoneResourceModel
	dataModel.CIDR = types.StringValue(cidr)
	dataModel.Name = types.StringValue(zoneName)
	dataModel.Kind = types.StringValue(zone.Kind)
	dataModel.ID = types.StringValue(zoneName)

	dataModel.Nameservers, _ = types.ListValueFrom(ctx, types.StringType, nameservers)

	resp.Diagnostics.Append(resp.State.Set(ctx, &dataModel)...)
}

func NewReverseZoneResource() resource.Resource {
	return &ReverseZoneResource{}
}
