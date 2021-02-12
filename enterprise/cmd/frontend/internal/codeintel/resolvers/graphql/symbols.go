package graphql

import (
	"context"
	"strings"

	"github.com/sourcegraph/go-lsp"
	gql "github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/graphqlbackend/graphqlutil"
	"github.com/sourcegraph/sourcegraph/enterprise/cmd/frontend/internal/codeintel/resolvers"
)

type DocSymbolConnectionResolver struct {
	symbols          []gql.DocSymbolResolver
	locationResolver *CachedLocationResolver
}

func NewDocSymbolConnectionResolver(symbols []*resolvers.AdjustedSymbol, locationResolver *CachedLocationResolver) gql.DocSymbolConnectionResolver {
	symbolResolvers := make([]gql.DocSymbolResolver, len(symbols))
	for i := range symbols {
		symbolResolvers[i] = newDocSymbolResolver(symbols[i], locationResolver)
	}
	return &DocSymbolConnectionResolver{symbols: symbolResolvers, locationResolver: locationResolver}
}

func (r *DocSymbolConnectionResolver) Nodes(ctx context.Context) ([]gql.DocSymbolResolver, error) {
	return r.symbols, nil
}

func (r *DocSymbolConnectionResolver) PageInfo(ctx context.Context) (*graphqlutil.PageInfo, error) {
	return graphqlutil.HasNextPage(false), nil
}

type docSymbolResolver struct {
	adjustedSymbol   *resolvers.AdjustedSymbol
	locationResolver *CachedLocationResolver
}

func newDocSymbolResolver(symbol *resolvers.AdjustedSymbol, locationResolver *CachedLocationResolver) *docSymbolResolver {
	return &docSymbolResolver{adjustedSymbol: symbol, locationResolver: locationResolver}
}

func (r *docSymbolResolver) ID(ctx context.Context) (string, error) {
	return r.adjustedSymbol.Identifier, nil
}

func (r *docSymbolResolver) Text(ctx context.Context) (string, error) {
	return r.adjustedSymbol.Text, nil
}

func (r *docSymbolResolver) Detail(ctx context.Context) (string, error) {
	return r.adjustedSymbol.Detail, nil
}
func (r *docSymbolResolver) Kind(ctx context.Context) (string, error) /* enum SymbolKind */ {
	// TODO(beyang): merge types (kludge)
	return strings.ToUpper(lsp.SymbolKind(r.adjustedSymbol.Kind).String()), nil
}
func (r *docSymbolResolver) Tags(ctx context.Context) ([]string, error) /* enum SymbolTag */ {
	tags := r.adjustedSymbol.Tags
	tagStrings := make([]string, len(tags))
	for i := range tags {
		tagStrings[i] = strings.ToUpper(tags[i].String())
	}
	return tagStrings, nil
}
func (r *docSymbolResolver) Definitions(ctx context.Context) (gql.LocationConnectionResolver, error) {
	// TODO(beyang): handle actual pagination
	adjustedLocations := make([]resolvers.AdjustedLocation, len(r.adjustedSymbol.AdjustedLocations))
	for i, loc := range r.adjustedSymbol.AdjustedLocations {
		adjustedLocations[i] = resolvers.AdjustedLocation{
			Dump:           r.adjustedSymbol.Dump,
			Path:           loc.Path,
			AdjustedCommit: loc.AdjustedCommit,
			AdjustedRange:  loc.AdjustedRange,
		}
	}
	return NewLocationConnectionResolver(adjustedLocations, nil, r.locationResolver), nil
}

func (r *docSymbolResolver) Children(ctx context.Context) ([]gql.DocSymbolResolver, error) {
	childrenResolvers := make([]gql.DocSymbolResolver, len(r.adjustedSymbol.Children))
	for i, child := range r.adjustedSymbol.Children {
		childrenResolvers[i] = newDocSymbolResolver(child, r.locationResolver)
	}
	return childrenResolvers, nil
}
