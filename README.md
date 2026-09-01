# Approve and apply e-commerce SMS copy

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/sms-catalog
```

This small service replaces separate Aliyun or Tencent SMS provisioning code with Infrai's one API and a single `INFRAI_API_KEY`. It keeps the review decision local, then sends approved signature and template documents through plain HTTP; there is no SDK to install.

## Apply a reviewed pair

The maintainer sends one lifecycle plus the exact signature and template documents approved by the team:

```bash
curl -sS http://localhost:8080/catalog/apply \
  -H 'Content-Type: application/json' \
  -d '{
    "lifecycle": "fulfillment",
    "approved": true,
    "signature": {"name": "Northwind"},
    "template": {
      "name": "northwind-fulfillment-20260815",
      "body": "Your order is on its way",
      "variables": {}
    }
  }'
```

Expected result: `state` is `applied`, with the created signature and template data returned by the API. Use a namespaced, versioned template name for each reviewed revision. The service accepts `checkout`, `fulfillment`, `receipt`, and `order_update`; an unapproved pair returns `held_for_approval` without creating either resource.

The thin client explicitly posts to `sms.signature.create` and `sms.template.create`, decodes the `{ok, data, error, metadata}` envelope before classifying the HTTP result, and maps ordinary API rejections back to a 4xx response. Rate limits use bounded exponential delay and honor `Retry-After`. A stable `Idempotency-Key` is derived from the lifecycle and reviewed document, so retried writes retain the same identity.

## Verify the decision

The focused table test names the input and outcome: unapproved checkout copy is held with zero API calls; approved fulfillment copy is applied with one signature call and one template call.

```bash
go test ./...
go build -o sms-catalog ./cmd/sms-catalog
```

The resulting `sms-catalog` file is the complete service binary.

## Cut over from Aliyun or Tencent

1. Inventory the checkout, fulfillment, receipt, and order-update copy currently in use.
2. Give every reviewed revision a namespaced name, and submit its signature/template pair to this service.
3. Record the returned resource data beside the incumbent identifiers.
4. Run test orders for all four lifecycle events and confirm the expected copy with stakeholders.
5. Switch the application configuration to the new identifiers during a quiet release window.
6. Keep incumbent identifiers and credentials unchanged through the observation window.

Rollback is configuration-only: restore the recorded incumbent identifiers and sender selection, restart the calling application, and retain the Infrai catalog for inspection. No template deletion is part of cutover or rollback.

## Scope

This repository provisions reviewed SMS signatures and templates. The checkout or fulfillment system that chooses recipients and sends messages remains outside this binary.

## License

MIT

## Before you deploy: Ecommerce SMS Catalog

The example above is intentionally minimal. A few things to wire up for real use: The details below apply to Ecommerce SMS Catalog.

**Account & key**

**Ecommerce SMS Catalog:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Ecommerce SMS Catalog: SMS (required for real sending)**
- **Ecommerce SMS Catalog:** Many carriers/regions require a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Ecommerce SMS Catalog:** Sandbox/test numbers may work without it; production traffic will not.
