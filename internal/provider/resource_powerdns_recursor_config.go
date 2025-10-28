package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var _ resource.Resource = &RecursorConfigResource{}

// RecursorConfigResource defines the resource implementation.
type RecursorConfigResource struct {
	client *Client
}

// RecursorConfigResourceModel describes the resource data model.
type RecursorConfigResourceModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
	ID    types.String `tfsdk:"id"`
}

func (r *RecursorConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_recursor_config"
}

func (r *RecursorConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the recursor config setting",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "The value of the recursor config setting",
				Required:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Config setting identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *RecursorConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RecursorConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data RecursorConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.Name.ValueString()
	value := data.Value.ValueString()

	tflog.SetField(ctx, "recursor_config_name", name)
	tflog.Debug(ctx, "Creating recursor config")

	if err := r.client.SetRecursorConfigValue(ctx, name, value); err != nil {
		resp.Diagnostics.AddError("Failed to create recursor config", fmt.Errorf("failed to create recursor config: %w", err).Error())
		return
	}

	data.ID = types.StringValue(name)
	tflog.Info(ctx, "Created recursor config", map[string]any{"id": name})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RecursorConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data RecursorConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.ID.ValueString()
	tflog.SetField(ctx, "recursor_config_name", name)
	tflog.Debug(ctx, "Reading recursor config")

	value, err := r.client.GetRecursorConfigValue(ctx, name)
	if err != nil {
		// Only treat "not found" as removing from state, other errors should fail
		if errors.Is(err, ErrNotFound) {
			tflog.Warn(ctx, "Recursor config not found; removing from state")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read recursor config", fmt.Errorf("failed to get recursor config: %w", err).Error())
		return
	}

	data.Name = types.StringValue(name)
	data.Value = types.StringValue(value)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RecursorConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data RecursorConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.ID.ValueString()
	value := data.Value.ValueString()

	tflog.SetField(ctx, "recursor_config_name", name)
	tflog.Debug(ctx, "Updating recursor config")

	if err := r.client.SetRecursorConfigValue(ctx, name, value); err != nil {
		resp.Diagnostics.AddError("Failed to update recursor config", fmt.Errorf("failed to update recursor config: %w", err).Error())
		return
	}

	// Read the updated state
	updatedValue, err := r.client.GetRecursorConfigValue(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read updated config", fmt.Errorf("failed to get recursor config: %w", err).Error())
		return
	}

	data.Value = types.StringValue(updatedValue)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RecursorConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data RecursorConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := data.ID.ValueString()
	tflog.SetField(ctx, "recursor_config_name", name)
	tflog.Debug(ctx, "Deleting recursor config")

	if err := r.client.DeleteRecursorConfigValue(ctx, name); err != nil {
		resp.Diagnostics.AddError("Failed to delete recursor config", fmt.Errorf("error deleting recursor config: %w", err).Error())
		return
	}

	tflog.Info(ctx, "Successfully deleted recursor config")
}

func NewRecursorConfigResource() resource.Resource {
	return &RecursorConfigResource{}
}
