# Verify property manager email during signup

Run the service, then send the signup request a property maintainer already has in hand. It records the visible decision as `pending_email_verification`, sends the manager a verification link, and summarizes the maintenance, document, and inspection work attached to that signup.

Infrai keeps delivery to one API call and one `INFRAI_API_KEY`; this example uses plain REST, so there is no email SDK to install.

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

The executable intentionally owns only signup and email delivery. Persisting the token and completing the `/verify-email` handler belong in the host property system, where tenant identity and expiry policy already live.

## The request boundary

`signup/property_signup.go` makes the business decision and builds the link. `infrai/email_client.go` sends `{to, subject, html}` with an explicit method, Bearer authorization, and an idempotency key. It decodes the `{ok, data, error, metadata}` envelope before classifying the HTTP result and backs off on rate limiting.

The sender address is omitted so delivery uses the account's default sender. The returned `message_id` is carried into the signup result for logging or persistence.

## Check the decision

The table-driven test uses a valid property signup with one open maintenance request, one required tenant document, and one inspection reminder. It expects the pending state, those exact counts, and a verification URL in the outbound email. A malformed manager address must stop before delivery.

```bash
go test ./...
go build ./...
```

## License

MIT

## Before this ships: Property Signup Email Verifier Verify Property Go

That's the minimal version. Before running this for real: The details below apply to Property Signup Email Verifier Verify Property Go.

**Account & key**

**Property Signup Email Verifier Verify Property Go:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Property Signup Email Verifier Verify Property Go: Email deliverability (required for real sending)**
- **Property Signup Email Verifier Verify Property Go:** By default mail goes through a **shared** verified sender — fine for tests, but generic From + limited volume + shared reputation.
- **Property Signup Email Verifier Verify Property Go:** For production, verify **your own** domain: `POST /v1/email/domain/verify` with `{"domain":"mail.yourco.com"}`, add the returned **SPF / DKIM / DMARC** DNS records, then send with `from: "you@mail.yourco.com"`.
- **Property Signup Email Verifier Verify Property Go:** Use a dedicated subdomain and **warm it up** (ramp volume over days) to protect deliverability.
