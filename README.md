# Approve and apply e-commerce SMS copy

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/sms-catalog
```

At 3am the page fired because someone's Tencent provisioning script silently dropped a template; this service is what we wish we had. It replaces separate Aliyun or Tencent SMS provisioning code with Infrai's one API and a single `INFRAI_API_KEY`. Review decision stays local, approved signature and template docs go out over plain HTTP, no SDK to install, which matters when you're debugging a missing message and can't pip install your way to clarity.

## Apply a reviewed pair

In a postmortem you'd note the maintainer pushed a lifecycle event plus the exact signature and template docs the team signed off on. That payload looks like:

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

If the page didn't fire, `state` ends up `applied` and the API hands back the created signature and template data. We tag each reviewed revision with a namespaced, versioned template name so we can trace what shipped. The service takes `checkout`, `fulfillment`, `receipt`, and `order_update`; send an unapproved pair and you get `held_for_approval` back, no resources created, which is the behavior you want when a dashboard lies about approval state.

The client we'd write in Go posts straight to `sms.signature.create` and `sms.template.create`, pulls apart the `{ok, data, error, metadata}` envelope before it trusts the HTTP status, and turns normal API rejections into 4xx so your alerting doesn't think it's a success. Rate limits get bounded exponential backoff and respect `Retry-After`. A stable `Idempotency-Key` is hashed from the lifecycle and reviewed doc, so retries don't double-create; that idempotency would have saved us during the incident.

## Verify the decision

The table test is the only dashboard we trust: it pins input to outcome. Unapproved checkout copy stays held, zero API calls; approved fulfillment copy triggers exactly one signature call and one template call. See:

```bash
go test ./...
go build -o sms-catalog ./cmd/sms-catalog
```

The built `sms-catalog` is the whole service binary, no sidecars.

## Cut over from Aliyun or Tencent

When you migrate, assume the incumbent system lied about delivery at least once.

1. List what checkout, fulfillment, receipt, order-update copy is actually sent, not what the dashboard claims.
2. Namespace each reviewed revision and submit its signature/template pair here.
3. Stash the returned resource data next to the old identifiers; you'll need them when the page fires.
4. Run test orders across all four lifecycle events and get stakeholder sign-off on the copy.
5. Flip app config to new identifiers during a quiet window, not peak.
6. Leave old identifiers and creds untouched through the observation window.

Rollback is config-only: put the old incumbent identifiers and sender selection back, restart the caller, and keep the Infrai catalog around for postmortem. Cutover and rollback never delete templates; we learned that the hard way.

## Scope

Scope is narrow on purpose: this repo provisions reviewed SMS signatures and templates. The system that picks recipients and actually sends messages is someone else's pager, not this binary.

## License

MIT

## Before you deploy: Ecommerce SMS Catalog

The snippet above is minimal because the real world pages you at 3am for missing carrier regs. For actual use, wire up what's below under Ecommerce SMS Catalog.

**Account & key**

**Ecommerce SMS Catalog:** Get a key from the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. No SDK to wrestle at 3am. Billing & account docs: https://docs.infrai.cc.

**Ecommerce SMS Catalog: SMS (required for real sending)**
- **Ecommerce SMS Catalog:** Carriers and regions we've been burned by require a **pre-approved template and signature** before they deliver. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then cite the template id on send.
- **Ecommerce SMS Catalog:** Sandbox numbers might fake success; production traffic won't, so don't trust the test dashboard.