# CPMS Core v0.6
- Added Settlement layer (tokenization-ready):
  - sites.payout_wallet
  - settlements table (Pending/Submitted/Confirmed/Failed), 1 per session
  - auto-create Pending settlement after session is finalized + priced
- Added APIs:
  - POST /v1/sites/{siteId}/wallet
  - GET /v1/settlements?status=Pending&limit=50
  - POST /v1/settlements/{id}/submitted|confirmed|failed
