# Acme API

Internal platform for Acme Corp — user management, team collaboration, and billing.

## Requirements

- **REQ-AUTH-001**: Users must authenticate via email/password or magic link
- **REQ-AUTH-002**: Sessions expire after 7 days of inactivity
- **REQ-AUTH-003**: Admin users can manage team members and roles
- **REQ-BILL-001**: Pro plan supports monthly and yearly billing via Stripe
- **REQ-BILL-002**: Usage is tracked per-team and enforced at API boundaries
- **REQ-BILL-003**: Downgrade must preserve data but restrict feature access
- **REQ-TEAM-001**: Teams have owners, admins, and members with distinct permissions
- **REQ-TEAM-002**: Team invitations sent via email with expiring tokens
- **REQ-API-001**: All mutations require authentication
- **REQ-API-002**: Rate limiting applies per-user at 100 requests/minute

## Architecture

Next.js API routes backed by Supabase (PostgreSQL + Auth), Stripe for billing, and Resend for transactional email. Middleware enforces auth and rate limits at the edge.

## Local Development

```bash
cp .env.example .env
npm install
npm run dev
```
