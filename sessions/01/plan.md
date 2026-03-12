# Plan: Namespace Serialization Deep-Dive & Decimal128

## ALL PHASES COMPLETE ✓

### Phase A: Decimal128 (⎕FR←1287) — DONE ✓
- Type byte confirmed as **0x2E** (not 0x2B)
- Activation: `⎕FR←1287` must be set BEFORE arithmetic (literals parsed before ⎕FR takes effect)
- 16-byte IEEE 754-2008 BID format
- Docs updated in `component-blocks.md`

### Phase B: Namespace Serialization — DONE ✓
- Sub-array stream format fully decoded: `[sw:u64][data:(sw-2)*8 bytes]`, advance=(sw-1)*8
- Four-section stream: NS_SELF_NAME → MEMBER_NAMES(reverse-sorted) → VALUES → INTERNAL_METADATA
- Name descriptors: 0x00=self, 0x28=variable, 0x98=child NS
- Per-scalar-member overhead: exactly 48 bytes (24 name + 24 value)
- Internal metadata: ~1752 bytes containing APL ⎕AV character translation tables + workspace state
- All 13 NS variants parsed successfully by `parse_ns_stream.py`

### Documentation — DONE ✓
- `docs/component-blocks.md`: Full namespace format spec + Decimal128 fix
- `docs/README.md`: Updated "Fully Documented" / "Partially Understood" / "Surprising Findings"

### Possible Future Work
- Decode namespace internal metadata type codes (0x14, 0x15, 0x1E)
- NS with functions (⎕FX) — dyascript doesn't support ∇ definitions
- NS_CHILD embedded format (sw=1 entry breaks current parser)
- NS with 10+ members to verify scaling
