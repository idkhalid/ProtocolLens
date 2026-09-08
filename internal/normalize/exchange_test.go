package normalize

import (
	"testing"

	"protocollens/internal/domain"
)

func TestEndpointOfDropsQuery(t *testing.T) {
	endpoint, err := EndpointOf(domain.Request{
		Method: "get",
		URL:    "https://Example.com/api/items?page=1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if endpoint.Method != "GET" || endpoint.Host != "example.com" || endpoint.Path != "/api/items" {
		t.Fatalf("endpoint = %#v", endpoint)
	}
}
