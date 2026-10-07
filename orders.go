package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// All money is integer cents to avoid floating-point rounding surprises.

type MenuItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PriceCents int    `json:"price_cents"`
}

var menu = map[string]MenuItem{
	"espresso":   {ID: "espresso", Name: "Espresso", PriceCents: 300},
	"cappuccino": {ID: "cappuccino", Name: "Cappuccino", PriceCents: 425},
	"latte":      {ID: "latte", Name: "Latte", PriceCents: 450},
	"cold-brew":  {ID: "cold-brew", Name: "Cold Brew", PriceCents: 400},
}

// discountCodes maps a code to a percentage off the subtotal.
var discountCodes = map[string]int{
	"WELCOME10": 10,
	"STAFF25":   25,
}

// taxBasisPoints is the sales tax rate in hundredths of a percent (8.75%).
const taxBasisPoints = 875

const maxQty = 20

// sizeAdjustCents is how much each drink size adds to the menu price.
// Medium is the menu price and the default when no size is given.
var sizeAdjustCents = map[string]int{
	"small":  -50,
	"medium": 0,
	"large":  75,
}

const defaultSize = "medium"

type OrderLine struct {
	ID   string `json:"id"`
	Qty  int    `json:"qty"`
	Size string `json:"size,omitempty"`
}

type OrderRequest struct {
	Items []OrderLine `json:"items"`
	Code  string      `json:"code,omitempty"`
}

type QuotedLine struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Qty       int    `json:"qty"`
	Size      string `json:"size"`
	UnitCents int    `json:"unit_cents"`
	LineCents int    `json:"line_cents"`
}

type Quote struct {
	Items         []QuotedLine `json:"items"`
	SubtotalCents int          `json:"subtotal_cents"`
	DiscountCents int          `json:"discount_cents"`
	TaxCents      int          `json:"tax_cents"`
	TotalCents    int          `json:"total_cents"`
}

var ErrEmptyOrder = errors.New("order has no items")

func sortedMenu() []MenuItem {
	items := make([]MenuItem, 0, len(menu))
	for _, m := range menu {
		items = append(items, m)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].PriceCents < items[j].PriceCents })
	return items
}

// quoteOrder prices an order: subtotal, then discount, then tax on the
// discounted amount.
func quoteOrder(req OrderRequest) (Quote, error) {
	if len(req.Items) == 0 {
		return Quote{}, ErrEmptyOrder
	}

	var q Quote
	for _, line := range req.Items {
		item, ok := menu[line.ID]
		if !ok {
			return Quote{}, fmt.Errorf("unknown item %q", line.ID)
		}
		if line.Qty < 1 || line.Qty > maxQty {
			return Quote{}, fmt.Errorf("qty for %q must be between 1 and %d", line.ID, maxQty)
		}
		size := line.Size
		if size == "" {
			size = defaultSize
		}
		adjust, ok := sizeAdjustCents[size]
		if !ok {
			return Quote{}, fmt.Errorf("unknown size %q for %q", line.Size, line.ID)
		}
		unitCents := item.PriceCents + adjust
		lineCents := unitCents * line.Qty
		q.Items = append(q.Items, QuotedLine{
			ID:        item.ID,
			Name:      item.Name,
			Qty:       line.Qty,
			Size:      size,
			UnitCents: unitCents,
			LineCents: lineCents,
		})
		q.SubtotalCents += lineCents
	}

	if req.Code != "" {
		pct, ok := discountCodes[strings.ToUpper(req.Code)]
		if !ok {
			return Quote{}, fmt.Errorf("unknown discount code %q", req.Code)
		}
		q.DiscountCents = q.SubtotalCents * pct / 100
	}

	taxable := q.SubtotalCents - q.DiscountCents
	q.TaxCents = (taxable*taxBasisPoints + 5000) / 10000 // round half up
	q.TotalCents = taxable + q.TaxCents
	return q, nil
}
