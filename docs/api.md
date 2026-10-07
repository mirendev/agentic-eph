# Brewbar API contract

This API is used by the web page in `static/index.html` and by the Brewbar
mobile app, which updates on its own schedule. Treat everything here as a
public contract:

- **Adding** endpoints or optional fields is fine.
- **Renaming, removing, or changing the type or meaning** of an existing field
  or endpoint is a breaking change. It needs a new versioned endpoint, not an
  edit in place.

All money values are integers in US cents.

## `GET /healthz`

Returns `200` with `{"status": "ok"}` when the app is up.

## `GET /api/menu`

```json
{
  "items": [
    { "id": "espresso", "name": "Espresso", "price_cents": 300 }
  ]
}
```

## `POST /api/orders`

Prices an order. Nothing is stored.

Request:

```json
{
  "items": [{ "id": "latte", "qty": 2 }],
  "code": "WELCOME10"
}
```

- `items` is required and must not be empty. `qty` is 1–20.
- `code` is optional and case-insensitive.

Response `200`:

```json
{
  "items": [
    { "id": "latte", "name": "Latte", "qty": 2, "unit_cents": 450, "line_cents": 900 }
  ],
  "subtotal_cents": 900,
  "discount_cents": 90,
  "tax_cents": 71,
  "total_cents": 881
}
```

Pricing rules:

1. `subtotal_cents` is the sum of `line_cents`.
2. `discount_cents` is the code's percentage of the subtotal, rounded down.
3. Tax is 8.75% of `subtotal_cents - discount_cents`, rounded half up.
4. `total_cents = subtotal_cents - discount_cents + tax_cents`.

Errors return a JSON body `{"error": "..."}`:

| Status | When |
|--------|------|
| `400` | Body is not valid JSON, or `items` is empty |
| `422` | Unknown item, `qty` out of range, or unknown discount code |
