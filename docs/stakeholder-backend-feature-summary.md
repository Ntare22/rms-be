# RMS-BE Backend Feature Summary

This document summarizes major backend capabilities in `rms-be` for stakeholder sharing.

## Implemented

### 1) Payments Module (Current)

- Payment initiation endpoint for creating pending placeholder payment transactions.
- Scope and authorization checks for organization, tenant ownership, and manager-assigned buildings.
- Designed for later payment gateway integration.

**Endpoint**
- `POST /api/v1/organizations/{id}/payments/initiate`

**Current data models**
- `payments.Payment`
- `payments.PaymentAllocation`

---

### 2) Lease and Tenant Operations

- Tenant lease decision workflow:
  - Approve lease
  - Reject lease
- Lease status support for:
  - `pending_approval`
  - `rejected`
- Unit occupancy synchronization after lease decisions and lifecycle updates.
- Manager building-scoped access enforcement across leases/tenants/units/buildings.
- Tenant role support in scoped lease and payment flows.

**Endpoints**
- `POST /api/v1/organizations/{id}/leases/{leaseId}/approve`
- `POST /api/v1/organizations/{id}/leases/{leaseId}/reject`

---

### 3) Tenant Invite and Password Setup

- Invite flow with one-time password setup token.
- Password setup request and confirm endpoints.
- Mailjet email sending with template variable support (Handlebars-style variables).

**Endpoints**
- `POST /api/v1/auth/password/setup/request`
- `POST /api/v1/auth/password/setup/confirm`

---

### 4) Billing Preferences and Reminder Workflow

- Billing preference fields on tenant and lease records.
- Worker process sends reminders for charges due in 7 days.
- Notification channel is selected by preference (email/SMS).
- Mailjet-based reminder email template support.

---

### 5) Access Control Improvements

- Organization-scoped lease and payment operations.
- Manager access restricted to assigned buildings.
- Active role support in current backend flows:
  - `admin`
  - `landlord`
  - `manager`
  - `staff`
  - `tenant`

---

### 6) Migration Coverage

Auto-migration includes newly introduced models used by current flows, including:

- `users.ManagerBuildingAssignment`
- `auth.PasswordSetupToken`
- existing payment/notification/audit models in active modules

## In Progress / Not Yet Implemented

The following items are not yet implemented in the current codebase:

- Payment summary endpoint
- Payment method split analytics endpoint
- Payment reminders candidates/history/send endpoints
- `payments.PaymentReminder` model
- Lease renewal workflow (offers, accept/reject, history)
- Lease closeout workflow
- Tenant statement and statement export endpoints
- `internal/integrations/pesapal` provider integration layer
- Additional roles: `property_manager`, `accountant`

## Notes

- Payment initiation currently creates a backend placeholder record and does not yet complete a live gateway checkout flow.
- This document summarizes functional backend capabilities and known gaps at this point in delivery.
