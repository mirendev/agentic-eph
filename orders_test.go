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
		{
			name: "small and large sizes",
			req: OrderRequest{Items: []OrderLine{
				{ID: "latte", Qty: 1, Size: "small"},
				{ID: "espresso", Qty: 2, Size: "large"},
			}},
			// 400 + 2*375 = 1150 subtotal, tax 100.625 -> 101
			want: Quote{SubtotalCents: 1150, TaxCents: 101, TotalCents: 1251},
		},
		{
			name: "explicit medium matches menu price",
			req:  OrderRequest{Items: []OrderLine{{ID: "latte", Qty: 2, Size: "medium"}}},
			want: Quote{SubtotalCents: 900, TaxCents: 79, TotalCents: 979},
		},
		{name: "unknown size", req: OrderRequest{Items: []OrderLine{{ID: "latte", Qty: 1, Size: "venti"}}}, wantErr: true},
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

func TestQuoteOrderLineSizes(t *testing.T) {
	got, err := quoteOrder(OrderRequest{Items: []OrderLine{
		{ID: "latte", Qty: 1},
		{ID: "latte", Qty: 1, Size: "small"},
		{ID: "latte", Qty: 2, Size: "large"},
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []QuotedLine{
		{ID: "latte", Name: "Latte", Qty: 1, Size: "medium", UnitCents: 450, LineCents: 450},
		{ID: "latte", Name: "Latte", Qty: 1, Size: "small", UnitCents: 400, LineCents: 400},
		{ID: "latte", Name: "Latte", Qty: 2, Size: "large", UnitCents: 525, LineCents: 1050},
	}
	if len(got.Items) != len(want) {
		t.Fatalf("got %d lines, want %d", len(got.Items), len(want))
	}
	for i := range want {
		if got.Items[i] != want[i] {
			t.Errorf("line %d = %+v, want %+v", i, got.Items[i], want[i])
		}
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
