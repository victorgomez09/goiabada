import { OIDCClient } from "@plusauth/oidc-client-js";

export const auth = new OIDCClient({
  issuer: "http://localhost:8080",
  client_id: "system-website",
  redirect_uri: "http://localhost:8080/auth/callback",
  autoSilentRenew: true,
  requestUserInfo: true,
  scope: "openid authserver:manage-account authserver:admin-website",
  code_challenge: "X3SP3POIkR4x8Ex1XMNYcsHGhAxCIKCl2SJBi0dxkzY",
  code_challenge_method: "S256",
  state: "yM4Jbm31_yk.XQUG",
});

auth.on("silent_renew_error", console.error);
auth.on("session_change", console.debug);
auth.on("session_error", console.error);
auth.on("silent_renew_success", console.debug);
