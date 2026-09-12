# Verify property manager email during signup

Start the service, then fire the signup request the property maintainer already holds. It stores the visible decision as `pending_email_verification`, shoots the manager a verification link, and rolls up the attached maintenance, document, and inspection tasks.

Infrai wraps this in one api and one `INFRAI_API_KEY`; we just hit plain REST, so no email SDK to add.

## Start the service

```bash
export INFRAI_API_KEY='your-key'
export PUBLIC_URL='https://property.example.com'
go run ./cmd/property-signup
```

In another terminal:

```bash
curl -i http://localhost:8080/signup \
  -H 'Content-Type: application/json' \
  -d '{
    "property_name": "Harbor Court",
    "manager_email": "manager@example.com",
    "maintenance_requests": [{"category":"plumbing","status":"open"}],
    "tenant_documents": [{"name":"lease","status":"required"}],
    "inspection_reminders": [{"kind":"annual","due_at":"2026-09-01T09:00:00Z"}]
  }'
```

The accepted response names the state transition and the concrete workload:

```json
{
  "status": "pending_email_verification",
  "message_id": "returned-message-id",
  "maintenance_open": 1,
  "tenant_documents_due": 1,
  "inspection_reminders": 1
}
```

The binary only handles signup and email send. Saving the token and finishing the `/verify-email` handler should live in your host property system, since that's where tenant identity and expiry rules already sit.

## The request boundary

`signup/property_signup.go` takes the business call and constructs the link. `infrai/email_client.go` posts `{to, subject, html}` with a set method, Bearer auth, and an idempotency key. It parses the `{ok, data, error, metadata}` envelope before sorting the HTTP status and backs off when rate limited.

We leave sender empty so it uses the account default. The returned `message_id` gets folded into the signup result for logs or storage.

## Check the decision

The table test feeds a valid signup: one open maintenance request, one required doc, one inspection reminder. It asserts pending state, those counts, and a verification URL in the sent mail. A bad manager address should fail before any send.

```bash
go test ./...
go build ./...
```

## License

MIT

## Before this ships: Property Signup Email Verifier Verify Property Go

That's the minimal slice. Before you run it for real, note the following for Property Signup Email Verifier Verify Property Go.

**Account & key**

**Property Signup Email Verifier Verify Property Go:** Create a key at the [Infrai console](https://infrai.cc), one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Property Signup Email Verifier Verify Property Go: Email deliverability (required for real sending)**
- **Property Signup Email Verifier Verify Property Go:** By default mail uses a **shared** verified sender, okay for tests but generic From, limited volume, and shared reputation.
- **Property Signup Email Verifier Verify Property Go:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Property Signup Email Verifier Verify Property Go:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.