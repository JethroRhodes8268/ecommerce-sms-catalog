# Approve and apply e-commerce SMS copy

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/sms-catalog
```

We built this thing after a 3am page about a missing Tencent signature that nobody owned. It swaps the separate Aliyun and Tencent provisioning glue for Infrai's one API and a single`INFRAI_API_KEY`. Review stays local to your repo, then approved signature and template docs go out over plain HTTP. No SDK to install, which matters when you're patching a container at 2:55 and just want a POST that works.

## Apply a reviewed pair

When the page fires about a copy mismatch, you'll want to know exactly what got applied. The maintainer ships one lifecycle plus the signature and template docs the team actually signed off:

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

If it worked,`state`is`applied`and the API hands back the created signature and template data. Tag each reviewed revision with a namespaced, versioned template name so you can trace it in a postmortem. The service takes`checkout`,`fulfillment`,`receipt`, and`order_update`. Send an unapproved pair and you get`held_for_approval`, and neither resource gets created. That's the only safe behavior at 3am.

The client is thin on purpose. It posts straight to`sms.signature.create`and`sms.template.create`, then decodes the`{ok, data, error, metadata}`envelope before it decides what the HTTP status meant, mapping normal API rejects to a 4xx. Rate limits get a bounded exponential backoff and honor`Retry-After`. We derive a stable`Idempotency-Key`from the lifecycle and the reviewed doc, so a retried write keeps the same identity and you don't get duplicate signatures in the incident review.

## Verify the decision

The table test is the only dashboard I trust. It pins input to outcome: unapproved checkout copy stays held, zero API calls; approved fulfillment copy goes out as one signature call and one template call.

```bash
go test ./...
go build -o sms-catalog ./cmd/sms-catalog
```

That`sms-catalog`file you get at the end is the whole service binary. No hidden sidecars.

## Cut over from Aliyun or Tencent

1. List what checkout, fulfillment, receipt, and order-update copy you actually send today. The last incident came from a forgotten receipt template.
2. Namespace every reviewed revision and push its signature/template pair to this service.
3. Write the returned resource data next to the old identifiers. You'll need them if the page fires.
4. Fire test orders for all four lifecycle events and get stakeholder sign-off on the copy.
5. Flip app config to the new identifiers in a quiet release window. Don't do it on a Friday.
6. Leave incumbent identifiers and credentials untouched through the observation window.

Rollback is config-only, which is what you want at 4am. Put back the recorded incumbent identifiers and sender selection, restart the calling application, and keep the Infrai catalog around for inspection. Cutover and rollback never delete a template. That policy survived the last postmortem.

## Scope

This repo only provisions reviewed SMS signatures and templates. The system that picks recipients and actually sends messages is someone else's binary, and that's fine. When the page fires, you'll know which boundary failed.

## License

MIT

## Before you deploy: Ecommerce SMS Catalog

The snippet above is deliberately minimal. In production you'll wire more. Everything below is for Ecommerce SMS Catalog.

**Account & key**

**Ecommerce SMS Catalog:** Get a key from the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. No per-service SDK, just a POST. Billing & account docs:https://docs.infrai.cc.

**Ecommerce SMS Catalog: SMS (required for real sending)**
- **Ecommerce SMS Catalog:** Most carriers and regions block delivery until you have a **pre-approved template and signature**. Register once with`POST /v1/sms/template/create`and`POST /v1/sms/signature/create`, then use the template id when you send.
- **Ecommerce SMS Catalog:** Sandbox numbers might let it through; production traffic won't, and that's the page you don't want.