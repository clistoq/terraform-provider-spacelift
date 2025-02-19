package spacelift

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/shurcooL/graphql"

	"github.com/spacelift-io/terraform-provider-spacelift/spacelift/internal"
	"github.com/spacelift-io/terraform-provider-spacelift/spacelift/internal/structs"
)

func resourceContext() *schema.Resource {
	return &schema.Resource{
		Description: "" +
			"`spacelift_context` represents a Spacelift **context** - " +
			"a collection of configuration elements (either environment variables or " +
			"mounted files) that can be administratively attached to multiple " +
			"stacks (`spacelift_stack`) or modules (`spacelift_module`) using " +
			"a context attachment (`spacelift_context_attachment`)`",

		CreateContext: resourceContextCreate,
		ReadContext:   resourceContextRead,
		UpdateContext: resourceContextUpdate,
		DeleteContext: resourceContextDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"description": {
				Type:        schema.TypeString,
				Description: "Free-form context description for users",
				Optional:    true,
			},
			"labels": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "Name of the context - should be unique in one account",
				Required:    true,
				ForceNew:    true,
			},
			"space_id": {
				Type:        schema.TypeString,
				Description: "ID (slug) of the space the context is in",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func resourceContextCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var mutation struct {
		CreateContext structs.Context `graphql:"contextCreate(name: $name, description: $description, labels: $labels, space: $space)"`
	}

	variables := map[string]interface{}{
		"name":        toString(d.Get("name")),
		"description": (*graphql.String)(nil),
		"labels":      (*[]graphql.String)(nil),
		"space":       (*graphql.ID)(nil),
	}

	if description, ok := d.GetOk("description"); ok {
		variables["description"] = toOptionalString(description)
	}

	if spaceID, ok := d.GetOk("space_id"); ok {
		variables["space"] = graphql.NewID(spaceID)
	}

	if labelSet, ok := d.Get("labels").(*schema.Set); ok {
		var labels []graphql.String

		for _, label := range labelSet.List() {
			labels = append(labels, graphql.String(label.(string)))
		}

		variables["labels"] = &labels
	}

	if err := meta.(*internal.Client).Mutate(ctx, "ContextCreate", &mutation, variables); err != nil {
		return diag.Errorf("could not create context: %v", internal.FromSpaceliftError(err))
	}

	d.SetId(mutation.CreateContext.ID)

	return resourceContextRead(ctx, d, meta)
}

func resourceContextRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var query struct {
		Context *structs.Context `graphql:"context(id: $id)"`
	}

	variables := map[string]interface{}{"id": graphql.ID(d.Id())}
	if err := meta.(*internal.Client).Query(ctx, "ContextRead", &query, variables); err != nil {
		return diag.Errorf("could not query for context: %v", err)
	}

	context := query.Context
	if context == nil {
		d.SetId("")
		return nil
	}

	d.Set("name", context.Name)

	if description := context.Description; description != nil {
		d.Set("description", *description)
	}

	labels := schema.NewSet(schema.HashString, []interface{}{})
	for _, label := range context.Labels {
		labels.Add(label)
	}
	d.Set("labels", labels)
	d.Set("space_id", context.Space)

	return nil
}

func resourceContextUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var mutation struct {
		UpdateContext structs.Context `graphql:"contextUpdate(id: $id, name: $name, description: $description, labels: $labels, space: $space)"`
	}

	variables := map[string]interface{}{
		"id":          toID(d.Id()),
		"name":        toString(d.Get("name")),
		"description": (*graphql.String)(nil),
		"labels":      (*[]graphql.String)(nil),
		"space":       (*graphql.ID)(nil),
	}

	if description, ok := d.GetOk("description"); ok {
		variables["description"] = toOptionalString(description)
	}

	if spaceID, ok := d.GetOk("space_id"); ok {
		variables["space"] = graphql.NewID(spaceID)
	}

	if labelSet, ok := d.Get("labels").(*schema.Set); ok {
		var labels []graphql.String

		for _, label := range labelSet.List() {
			labels = append(labels, graphql.String(label.(string)))
		}

		variables["labels"] = &labels
	}

	var ret diag.Diagnostics

	if err := meta.(*internal.Client).Mutate(ctx, "ContextUpdate", &mutation, variables); err != nil {
		ret = append(ret, diag.Errorf("could not update context: %v", internal.FromSpaceliftError(err))...)
	}

	ret = append(ret, resourceContextRead(ctx, d, meta)...)

	return ret
}

func resourceContextDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var mutation struct {
		DeleteContext *structs.Context `graphql:"contextDelete(id: $id)"`
	}

	variables := map[string]interface{}{"id": toID(d.Id())}

	if err := meta.(*internal.Client).Mutate(ctx, "ContextDelete", &mutation, variables); err != nil {
		return diag.Errorf("could not delete context: %v", internal.FromSpaceliftError(err))
	}

	d.SetId("")

	return nil
}
