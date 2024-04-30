package variables

import (
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
)

const (
	// varBlockName is the name of the custom block type that allows the
	// definition of custom variables.
	varBlockName = "variables"

	// varObjectName is the name of the object that is used to reference
	// variables within the "variables" block.
	varObjectName = "var"
)

// Variables is a custom block type that allows the definition of custom
// variables within the "variables" block. These variables can be referenced
// globally through the "var" object. Additionally, variables can reference
// each other within the "variables" block itself.
//
// Example:
//
//	variables {
//	  var "foo" {
//	    value = "bar"
//	  }
//	}
//
//	block "example" {
//	  foo = var.foo.value
//	}
//
// NOTE: If some variable was defined multiple times within the body (in different
// variable blocks), then the last definition will be stored in a context.
func Variables(ctx *hcl.EvalContext, body hcl.Body) (hcl.Body, hcl.Diagnostics) {
	content, remain, diags := body.PartialContent(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{{Type: varBlockName}},
	})
	if diags.HasErrors() {
		return nil, diags
	}

	attrs, diags := collectAttributes(content)
	if diags.HasErrors() {
		return nil, diags
	}

	vars, diags := attrs2variables(ctx, attrs)
	if diags.HasErrors() {
		return nil, diags
	}

	if ctx.Variables == nil {
		ctx.Variables = make(map[string]cty.Value)
	}
	ctx.Variables[varObjectName] = cty.ObjectVal(make(map[string]cty.Value))

	for i := range vars {
		diags = diags.Extend(vars[i].Evaluate(ctx))
	}
	return remain, diags
}

func collectAttributes(content *hcl.BodyContent) (attrs hcl.Attributes, diags hcl.Diagnostics) {
	attrs = make(hcl.Attributes)
	for _, block := range content.Blocks {
		battrs, bdiags := block.Body.JustAttributes()
		if diags.HasErrors() {
			diags = diags.Extend(bdiags)
			continue
		}
		maps.Copy(attrs, battrs)
	}
	return attrs, diags
}

func attrs2variables(ctx *hcl.EvalContext, attrs hcl.Attributes) ([]*variable, hcl.Diagnostics) {
	variables := collectVariables(attrs)
	return topologicalSort(ctx, variables)
}

func collectVariables(attrs hcl.Attributes) []*variable {
	l := make([]*variable, 0, len(attrs))
	m := make(map[string]*variable)
	for _, attr := range attrs {
		name := globalName(attr.Name)
		node, ok := m[name]
		if ok {
			node.Attr = attr
		} else {
			node = &variable{Name: name, Attr: attr}
			l = append(l, node)
			m[name] = node
		}

		for _, v := range attr.Expr.Variables() {
			refNames := traversal2names(v)
			for _, name := range refNames {
				// if we have self-references, then we need to retry
				// evaluation of the variable to resolve it.
				if name == node.Name && len(refNames) > 1 {
					node.Retry++
					continue
				}
				ref, ok := m[name]
				if ok {
					node.Reference = append(node.Reference, ref)
				} else {
					ref = &variable{Name: name}
					node.Reference = append(node.Reference, ref)
					l = append(l, ref)
					m[name] = ref
				}
			}
		}
	}
	// to have consistent results in tests, we need to keep the order
	// of attributes.
	sort.Slice(l, func(i, j int) bool { return l[i].Name < l[j].Name })
	return l
}

func globalName(name string) string {
	if !strings.HasPrefix(name, varObjectName+".") {
		return varObjectName + "." + name
	}
	return name
}

func traversal2names(tr hcl.Traversal) []string {
	if tr.IsRelative() {
		panic("unexpected relative traversal")
	}
	var l []string
	name := ""
	for i := range tr {
		switch t := tr[i].(type) {
		case hcl.TraverseRoot:
			name = t.Name
			if name != varObjectName {
				name = varObjectName + "." + name
				l = append(l, name)
			}
		case hcl.TraverseAttr:
			name += "." + t.Name
			l = append(l, name)
		case hcl.TraverseIndex:
			name += "." + t.Key.GoString()
			l = append(l, name)
		case hcl.TraverseSplat:
			// TODO: I have no idea what splat
			// is and have never seen anything
			// like this in our configs
			panic("unexpected traversal type: splat")
		default:
			panic(fmt.Sprintf("unexpected traversal type: %T", t))
		}
	}
	return l
}

// topologicalSort using DFS algorithm (https://en.wikipedia.org/wiki/Topological_sorting)
func topologicalSort(ctx *hcl.EvalContext, nodes []*variable) ([]*variable, hcl.Diagnostics) {
	if len(nodes) == 0 {
		return nil, nil
	}

	var res []*variable
	temp := make(map[string]bool, len(nodes))
	mark := make(map[string]bool, len(nodes))

	var visit func(node *variable) hcl.Diagnostics
	visit = func(node *variable) hcl.Diagnostics {
		if mark[node.Name] {
			return nil
		}
		if temp[node.Name] {
			return hcl.Diagnostics{{
				Severity:    hcl.DiagError,
				Summary:     "Circular reference detected",
				Detail:      "Variable refers to itself through a circular reference.",
				Subject:     node.Attr.Expr.Range().Ptr(),
				Expression:  node.Attr.Expr,
				EvalContext: ctx,
			}}
		}
		temp[node.Name] = true
		for _, ref := range node.Reference {
			diags := visit(ref)
			if diags.HasErrors() {
				return diags
			}
		}
		temp[node.Name] = false
		mark[node.Name] = true

		res = append(res, node)
		return nil
	}

	for _, node := range nodes {
		diags := visit(node)
		if diags.HasErrors() {
			return nil, diags
		}
	}
	return res, nil
}

type variable struct {
	Name      string
	Attr      *hcl.Attribute
	Reference []*variable
	Retry     int
}

func (n *variable) Evaluate(ctx *hcl.EvalContext) hcl.Diagnostics {
	if n.Attr == nil {
		return nil
	}

	diags := n.evaluate(ctx)
	for i := 0; i < n.Retry; i++ {
		diags = n.evaluate(ctx)
	}
	return diags
}

func (n *variable) evaluate(ctx *hcl.EvalContext) hcl.Diagnostics {
	value, diags := n.Attr.Expr.Value(ctx)
	setContextVariable(ctx, n.Attr.Name, value)
	return diags
}

func setContextVariable(ctx *hcl.EvalContext, name string, value cty.Value) {
	object := ctx.Variables[varObjectName]
	if object.IsNull() {
		ctx.Variables[varObjectName] = cty.ObjectVal(
			map[string]cty.Value{
				name: value,
			})
		return
	}
	values := object.AsValueMap()
	if values == nil {
		ctx.Variables[varObjectName] = cty.ObjectVal(
			map[string]cty.Value{
				name: value,
			})
		return
	}

	values[name] = value
	ctx.Variables[varObjectName] = cty.ObjectVal(values)
}
