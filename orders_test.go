package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestQuoteOrder(t *testing.T) {
	tests := []struct {
		name    string
		req     OrderRequest
		want    Quote
		wantErr bool
	}{
		{
			name: "single item",
			req:  OrderRequest{Items: []OrderLine{{ID: "latte", Qty: 2}}},
			// 900 subtotal, tax 78.75 -> 79
			want: Quote{SubtotalCents: 900, TaxCents: 79, TotalCents: 979},
		},
		{
			name: "discount applied before tax",
			req: OrderRequest{
				Items: []OrderLine{{ID: "espresso", Qty: 1}, {ID: "cold-brew", Qty: 1}},
				Code:  "welcome10",
			},
			// 700 subtotal, 70 off, 630 taxable, tax 55.125 -> 55
			want: Quote{SubtotalCents: 700, DiscountCents: 70, TaxCents: 55, TotalCents: 685},
		},
		{name: "empty", req: OrderRequest{}, wantErr: true},
		{name: "unknown item", req: OrderRequest{Items: []OrderLine{{ID: "tea", Qty: 1}}}, wantErr: true},
		{name: "zero qty", req: OrderRequest{Items: []OrderLine{{ID: "latte", Qty: 0}}}, wantErr: true},
		{name: "bad code", req: OrderRequest{Items: []OrderLine{{ID: "latte", Qty: 1}}, Code: "FREE"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := quoteOrder(tt.req)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.SubtotalCents != tt.want.SubtotalCents ||
				got.DiscountCents != tt.want.DiscountCents ||
				got.TaxCents != tt.want.TaxCents ||
				got.TotalCents != tt.want.TotalCents {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestOrdersEndpoint(t *testing.T) {
	srv := httptest.NewServer(newMux())
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/orders", "application/json",
		strings.NewReader(`{"items":[{"id":"latte","qty":1}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}
