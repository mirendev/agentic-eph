package main

import (
	"encoding/json"
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
		{
			name: "large adds 75 cents",
			req:  OrderRequest{Items: []OrderLine{{ID: "latte", Qty: 1, Size: "large"}}},
			// 525 subtotal, tax 45.9375 -> 46
			want: Quote{SubtotalCents: 525, TaxCents: 46, TotalCents: 571},
		},
		{
			name: "small takes off 50 cents",
			req:  OrderRequest{Items: []OrderLine{{ID: "espresso", Qty: 2, Size: "small"}}},
			// 2 x 250 = 500 subtotal, tax 43.75 -> 44
			want: Quote{SubtotalCents: 500, TaxCents: 44, TotalCents: 544},
		},
		{
			name: "mixed sizes with discount",
			req: OrderRequest{
				Items: []OrderLine{
					{ID: "cappuccino", Qty: 1, Size: "small"},
					{ID: "cold-brew", Qty: 2, Size: "medium"},
					{ID: "latte", Qty: 1, Size: "large"},
				},
				Code: "STAFF25",
			},
			// 375 + 800 + 525 = 1700 subtotal, 425 off, 1275 taxable,
			// tax 111.5625 -> 112
			want: Quote{SubtotalCents: 1700, DiscountCents: 425, TaxCents: 112, TotalCents: 1387},
		},
		{name: "unknown size", req: OrderRequest{Items: []OrderLine{{ID: "latte", Qty: 1, Size: "venti"}}}, wantErr: true},
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

	var body struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if len(body.Items) != 1 || body.Items[0]["size"] != "medium" {
		t.Errorf("items = %v, want one line with size medium", body.Items)
	}
}

func TestOrdersEndpointUnknownSize(t *testing.T) {
	srv := httptest.NewServer(newMux())
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/orders", "application/json",
		strings.NewReader(`{"items":[{"id":"latte","qty":1,"size":"venti"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
}

func TestQuotedLineSize(t *testing.T) {
	got, err := quoteOrder(OrderRequest{Items: []OrderLine{
		{ID: "latte", Qty: 1},
		{ID: "latte", Qty: 3, Size: "small"},
		{ID: "espresso", Qty: 2, Size: "large"},
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []QuotedLine{
		{ID: "latte", Name: "Latte", Qty: 1, Size: "medium", UnitCents: 450, LineCents: 450},
		{ID: "latte", Name: "Latte", Qty: 3, Size: "small", UnitCents: 400, LineCents: 1200},
		{ID: "espresso", Name: "Espresso", Qty: 2, Size: "large", UnitCents: 375, LineCents: 750},
	}
	if len(got.Items) != len(want) {
		t.Fatalf("got %d lines, want %d", len(got.Items), len(want))
	}
	for i := range want {
		if got.Items[i] != want[i] {
			t.Errorf("line %d: got %+v, want %+v", i, got.Items[i], want[i])
		}
	}
}
