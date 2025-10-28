package provider

import (
	"context"
	"errors"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var _ resource.Resource = &RecursorForwardZoneResource{}

// RecursorForwardZoneResource defines the resource implementation.
type RecursorForwardZoneResource struct {
	client *Client
}

// RecursorForwardZoneResourceModel describes the resource data model.
type RecursorForwardZoneResourceModel struct {
	Zone    types.String `tfsdk:"zone"`
	Servers types.List   `tfsdk:"servers"`
	ID      types.String `tfsdk:"id"`
}

func (r *RecursorForwardZoneResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_recursor_forward_zone"
}

func (r *RecursorForwardZoneResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"zone": schema.StringAttribute{
				MarkdownDescription: "The zone name to forward",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"servers": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "List of DNS servers to forward queries to",
				Required:            true,
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

func (r *RecursorForwardZoneResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RecursorForwardZoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RecursorForwardZoneResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone := data.Zone.ValueString()
	var servers []string
	if !data.Servers.IsNull() {
		for _, s := range data.Servers.Elements() {
			if str, ok := s.(types.String); ok {
				servers = append(servers, str.ValueString())
			}
		}
	}

	tflog.SetField(ctx, "zone", zone)
	tflog.Debug(ctx, "Creating recursor forward zone")

	// Get current forward-zones
	currentValue, err := r.client.GetRecursorConfigValue(ctx, "forward-zones")
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			currentValue = ""
		} else {
			resp.Diagnostics.AddError("Failed to get current forward-zones config", err.Error())
			return
		}
	}

	// Parse current forward-zones
	forwardZones := parseForwardZones(currentValue)

	// Add/update zone
	forwardZones[zone] = servers

	// Serialize back
	newValue := serializeForwardZones(forwardZones)

	if err := r.client.SetRecursorConfigValue(ctx, "forward-zones", newValue); err != nil {
		resp.Diagnostics.AddError("Failed to create recursor forward zone", err.Error())
		return
	}

	data.ID = types.StringValue(zone)
	tflog.Info(ctx, "Created recursor forward zone", map[string]any{"id": zone})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RecursorForwardZoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RecursorForwardZoneResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone := data.ID.ValueString()

	tflog.SetField(ctx, "zone", zone)
	tflog.Debug(ctx, "Reading recursor forward zone")

	value, err := r.client.GetRecursorConfigValue(ctx, "forward-zones")
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, "Recursor forward-zones config not found; removing from state")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to get forward-zones config", err.Error())
		return
	}

	forwardZones := parseForwardZones(value)

	servers, exists := forwardZones[zone]
	if !exists {
		tflog.Warn(ctx, "Forward zone not found; removing from state")
		resp.State.RemoveResource(ctx)
		return
	}

	data.Zone = types.StringValue(zone)
	var serversList []types.String
	for _, s := range servers {
		serversList = append(serversList, types.StringValue(s))
	}
	data.Servers, _ = types.ListValueFrom(ctx, types.StringType, serversList)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RecursorForwardZoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RecursorForwardZoneResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone := data.ID.ValueString()
	var servers []string
	if !data.Servers.IsNull() {
		for _, s := range data.Servers.Elements() {
			if str, ok := s.(types.String); ok {
				servers = append(servers, str.ValueString())
			}
		}
	}

	tflog.SetField(ctx, "zone", zone)
	tflog.Debug(ctx, "Updating recursor forward zone")

	// Get current forward-zones
	currentValue, err := r.client.GetRecursorConfigValue(ctx, "forward-zones")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get current forward-zones", err.Error())
		return
	}

	// Parse current forward-zones
	forwardZones := parseForwardZones(currentValue)

	// Update zone
	forwardZones[zone] = servers

	// Serialize back
	newValue := serializeForwardZones(forwardZones)

	if err := r.client.SetRecursorConfigValue(ctx, "forward-zones", newValue); err != nil {
		resp.Diagnostics.AddError("Failed to update recursor forward zone", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RecursorForwardZoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RecursorForwardZoneResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	zone := data.ID.ValueString()

	tflog.SetField(ctx, "zone", zone)
	tflog.Debug(ctx, "Deleting recursor forward zone")

	// Get current forward-zones
	currentValue, err := r.client.GetRecursorConfigValue(ctx, "forward-zones")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get current forward-zones", err.Error())
		return
	}

	// Parse current forward-zones
	forwardZones := parseForwardZones(currentValue)

	// Remove zone
	delete(forwardZones, zone)

	// Serialize back
	newValue := serializeForwardZones(forwardZones)

	if err := r.client.SetRecursorConfigValue(ctx, "forward-zones", newValue); err != nil {
		resp.Diagnostics.AddError("Error deleting recursor forward zone", err.Error())
		return
	}

	tflog.Info(ctx, "Successfully deleted recursor forward zone")
}

func (r *RecursorForwardZoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func NewRecursorForwardZoneResource() resource.Resource {
	return &RecursorForwardZoneResource{}
}

// parseForwardZones parses the forward-zones string into a map.
func parseForwardZones(value string) map[string][]string {
	result := make(map[string][]string)
	if value == "" {
		return result
	}

	entries := strings.Split(value, ";")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			zone := strings.TrimSpace(parts[0])
			serversStr := strings.TrimSpace(parts[1])
			servers := strings.Split(serversStr, ",")
			for i, s := range servers {
				servers[i] = strings.TrimSpace(s)
			}
			result[zone] = servers
		}
	}
	return result
}

// serializeForwardZones serializes the map back to forward-zones string.
func serializeForwardZones(zones map[string][]string) string {
	var entries []string
	for zone, servers := range zones {
		entries = append(entries, zone+"="+strings.Join(servers, ","))
	}
	return strings.Join(entries, ";")
}
