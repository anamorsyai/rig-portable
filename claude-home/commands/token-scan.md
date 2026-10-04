---
description: "Fast rug-pull / token-security audit for meme coin or token contracts (EVM Solidity or Solana Rust/Anchor all supported). Checks hidden mint, honeypot, fee manipulation, LP-lock bypass, authority retention, bonding-curve exploits, fake renounce, sandwich amplification. Manual 8-class grep audit; optional token_scanner.py accelerates. Usage: /token-scan <contract_path_or_dir> [--chain solana] [--recursive]"
---

# TOKEN-SCAN — $ARGUMENTS

Self-contained manual 8-class rug-pull audit. No external scanner required — the
grep audit below is the real check. An optional `token_scanner.py` on PATH only
accelerates it; if absent, skip straight to Step 2.

## Step 0 — Quick Kill Signals

```
[ ] Contract is verified (source code available)?
[ ] Deployer has no rug history?
[ ] Token has been trading > 1 hour?
[ ] Liquidity > $5K?
[ ] Not a proxy with retained admin?
```

If ANY is NO → flag and proceed with extreme caution.

## Step 1 — Optional accelerated scanner

> Skip if `token_scanner.py` is not on PATH — not bundled with this rig.

```
python3 token_scanner.py <contract>        # EVM
python3 token_scanner.py <dir> --chain solana --recursive   # Solana
```

## Step 2 — Manual 8-class grep audit

For EVM (Solidity) and Solana (Rust/Anchor), audit each class:

1. **Hidden mint** — `_mint(` / `mint_to(` / `mint_to_internal(` reachable by
   non-restricted caller; search for missing `onlyOwner`/authority check around call sites.
2. **Honeypot** — `_transfer(` includes a `buyer`/`seller`/`address set` check that
   blocks the buyer's sell; `isExcludedFromFees` used as a whitelist; check both
   buy+ sell paths, then whether excluded users can sell.
3. **Fee manipulation** — `setFees(` / `updateFees(` with no max cap → owner can
   drain via 100% sell fee; check `buyFee`/`sellFee` setters for a `<=` bound.
4. **LP lock bypass** — `migrateLiquidity(` / `sync(` / `rescueTokens(` that can
   pull LP tokens out of the pair; check for a `renounceOwnership` that leaves
   LP still removable.
5. **Authority retention** — proxy/upgradeable pattern where the admin/authority
   was neither renounced nor burned; multi-sig false-renounce.
6. **Bonding curve exploits** — buy/sell price calc with rounding in the user's
   favor; missing slippage bound on sell; check `getAmountOut` divergence between
   swap and direct transfer.
7. **Fake renounce** — `renounceOwnership()` present but only transfers to a
   zero address while `_owner` setter still reachable; or ownership was renounced
   but a `pendingOwner`/`timelock` still retains control.
8. **Sandwich amplification** — fees or amounts that amplify a MEV sandwich
   (liquidity ratio manipulation pre-trade); check `getReserves` assumptions.

For each class, output: `[FOUND|CLEAR] class — <evidence: file:line + snippet>`.

## Step 3 — Report

Save verdict to `findings/token-audit-<contract>.md` with per-class evidence.

---

*Sourced from the claude-bughunter bundle; adapted to rig command format. The
core /report, /verify, /skeptic flow owns any downstream finding.*
