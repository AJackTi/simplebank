# Authentication token cutover

The authentication claims are intentionally versioned by purpose. New access
and refresh tokens carry an explicit `token_type`, issuer, and audience, and
the API verifies all three values before accepting a token.

This is a deliberate security cutover. Tokens issued before this change do
not contain those claims and are rejected. Users must sign in again after the
deployment to obtain new access and refresh tokens. Existing database session
rows are harmless until a new token is presented and can be removed by the
normal session-retention policy.

Operators should announce the forced re-authentication window before rollout
and monitor login failures during the cutover. Do not loosen validation to
accept legacy tokens: an old token cannot safely identify whether it is being
used as an access token or a refresh token.
