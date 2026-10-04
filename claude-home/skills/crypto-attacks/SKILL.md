---
name: crypto-attacks
description: "Cryptographic attacks: weak algorithms, padding oracle, timing attacks, key leakage, CBC bit flipping."
---

# Crypto Attacks

## Weak Algorithms
- MD5/SHA1 collision attacks
- DES (56-bit key) brute force
- RC4 bias exploitation
- ECB mode detection (identical blocks = identical ciphertext)

## Padding Oracle
- Send encrypted data with modified last byte
- 200/500 diff reveals padding validity
- Decrypt any ciphertext without key
- Tools: PadBuster, padoracle

## Timing Attacks
- Measure response time differences
- HMAC comparison timing
- RSA PKCS#1 v1.5 padding oracle
- Use t-test for statistical significance

## CBC Bit Flipping
- Modify IV to change first plaintext block
- Modify ciphertext block to affect next block
- XOR calculated value into previous ciphertext block

## Key Leakage
- Check for hardcoded keys in source
- Check for key reuse across environments
- Check for weak key generation (low entropy)
- JWT none algorithm attack
