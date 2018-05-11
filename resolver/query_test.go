package resolver_test

import (
	"testing"

	"github.com/kevinclcn/graphql-go-example/resolver"
	"github.com/kevinclcn/graphql-go-example/schema"

	graphql "github.com/graph-gophers/graphql-go"
)

func TestResolversSatisfySchema(t *testing.T) {
	rootResolver := &resolver.QueryResolver{}
	_, err := graphql.ParseSchema(schema.String(), rootResolver)
	if err != nil {
		t.Error(err)
	}
}
