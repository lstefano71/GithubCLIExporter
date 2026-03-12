<overview>
The user is reverse-engineering the Dyalog APL Component File (DCF) binary format, specifically the internal array serialization used by `220⌶` (Serialise Array). This session focused on three things: (1) fixing bugs in the Python analyzer for matrix/nested/namespace parsing, (2) confirming Decimal128 (⎕DR 1287) serialization by properly activating it via `⎕FR←1287` with arithmetic (not literals), and (3) deep investigation of namespace serialization format through controlled differential binary analysis. The approach is experimental: create APL scripts that serialize controlled test values, run them via `dyascript.exe`, then analyze the binary output with Python.
</overview>

<history>
1. Session started with fixing bugs in `analyze_serialization.py`
   - Fixed matrix/higher-rank shape parsing — shape dimensions now read from correct offset (starting at `element_count` field position for rank≥2)
   - Added nested array detection (rank byte low nibble 0x07 vs 0x0F for simple)
   - Added namespace type recognition (type_code 0x0000)
   - Added Complex128 scalar display, Char32 decoding, boolean MSB-first unpacking
   - Verified all 57 test components across 6 DCF files parse correctly

2. Updated documentation with complete array serialization findings
   - Updated `docs/component-blocks.md` with: complete type byte mapping (all 11+ types), two rank byte patterns (simple 0x0F vs nested 0x07), shape encoding for rank≥2, boolean MSB-first packing, integer promotion boundaries, nested array recursive format, namespace serialization notes, 220⌶↔DCF block correspondence
   - Updated `docs/README.md`: moved items from "Not Yet Investigated" to "Fully Documented", added new surprising findings, added test file inventory

3. User asked to tackle namespace serialization and Decimal128 activation (pointing to ⎕FR docs)
   - Fetched ⎕FR docs — key insight: `⎕FR←1287` only affects NEW computations, not literal constants
   - Created plan for Decimal128 fix and namespace deep-dive

4. Created and ran `probe_decimal128.apls`
   - Used arithmetic (`1÷3`, `(⍳5)÷10`) instead of literals after `⎕FR←1287`
   - **KEY DISCOVERY: Decimal128 type byte = 0x2E (46), NOT 0x2B as hypothesized!**
   - ⎕DR confirmed as 1287 for computed values
   - Scalar: 34 bytes (16-byte IEEE 754-2008 BID format value)
   - Vector of 5: 106 bytes (16 bytes per element)
   - Verified BID encoding: 0.1 = coefficient=1, biased_exponent=6175 (exponent=-1). 14-bit exponent in bits 126-113, 113-bit coefficient in bits 112-0
   - DCF creation failed in script (file path issue) but serialization data captured

5. Created and ran `probe_ns_deep.apls` — 13 controlled namespace variants
   - Empty NS, 1/2/3 int members, long name, float/string/nested/sub-NS/5-mixed members, different name lengths
   - Had to rewrite without `∇` function definitions (dyascript doesn't support them)
   - Fixed `⎕NDELETE` error by wrapping in `:Trap`
   - All 13 dumps captured successfully, DCF file with all 13 components created
   - **KEY SIZE FINDINGS:**
     - Empty NS: 1986 bytes
     - Each member adds exactly 48 bytes (2034, 2082, 2130 for 1/2/3 members)
     - Names 'a', 'ab', 'abc' all produce same size (2034) — short names fit in 8-byte slot
     - 'longname' (8 chars): 2050 = 2034 + 16 extra bytes
     - Float64 member same size as Int8 (2034) — scalar values always use 8 bytes
     - String 'hello': 2042 (+8 from scalar)
     - Nested array: 2122 (+88 from scalar)
     - Sub-namespace: 2274 (embeds full NS overhead)

6. Created `analyze_ns.py` — Python binary analysis/diffing tool
   - Parses DUMP output from probe, finds UTF-16 strings, finds type codes, does byte-level diffs
   - Ran successfully, output captured in `ns_analysis_output.txt`
   - **Was in the middle of analyzing the output when compaction occurred**

</history>

<work_done>
Files created:
- `probe_decimal128.apls`: Decimal128 experiment using ⎕FR←1287 with arithmetic. Confirmed type byte 0x2E.
- `probe_ns_deep.apls`: 13 controlled namespace variants for differential analysis. All ran successfully.
- `analyze_ns.py`: Python tool for namespace binary analysis — parses dumps, finds strings/type codes, does diffs.
- `test_ns_variants.dcf`: DCF file with 13 namespace components (empty, 1-3 members, various types).
- `ns_probe_output.txt`: Raw output from probe_ns_deep.apls (all 13 byte dumps as decimal values).
- `ns_analysis_output.txt`: Output from analyze_ns.py (large file, ~291KB).

Files modified:
- `analyze_serialization.py`: Fixed shape parsing, nested detection, namespace type, Char32/Complex128/Bool display.
- `docs/component-blocks.md`: Major update — complete type mapping, rank patterns, shape encoding, nested format, namespace notes, 220⌶ correspondence.
- `docs/README.md`: Updated "Fully Documented" / "Not Yet Investigated" sections, added test files, added surprising findings.

Work completed:
- [x] Fix analyze_serialization.py matrix/nested/namespace bugs
- [x] Update documentation with complete array serialization findings
- [x] Confirm Decimal128 type byte (0x2E) via ⎕FR←1287 + arithmetic
- [x] Create and run NS probe with 13 controlled variants
- [x] Create NS binary analyzer tool
- [ ] **IN PROGRESS**: Analyze ns_analysis_output.txt to map namespace internal structure
- [ ] Document namespace serialization format in docs
- [ ] Update docs with Decimal128 type byte correction (0x2E not 0x2B)

SQL todos status:
- decimal128: done
- ns-probe: done  
- ns-analyze: in_progress
- ns-docs: pending (depends on ns-analyze)
</work_done>

<technical_details>
## Decimal128 (⎕DR 1287)
- **Type byte = 0x2E** (decimal 46), NOT 0x2B as previously hypothesized
- Activation: `⎕FR←1287` must be set BEFORE computation. Literal constants are parsed before `⎕FR` takes effect on the same line. Use arithmetic: `x←1÷3` or `x←0+1.5`
- Element size: 16 bytes, stored in IEEE 754-2008 BID (Binary Integer Decimal) format
- BID format: sign(1 bit, 127) + exponent(14 bits, 126-113, biased by 6176) + coefficient(113 bits, 112-0, as binary integer)
- Verified: 0.1 = coefficient=1, exponent=-1 (biased=6175); 0.2 = coefficient=2, exponent=-1
- The scalar 1÷3 = 0x0219a45894e48295_67d9da2155555555 (repeating pattern visible)

## Namespace Serialization Structure (Partial — analysis in progress)
From raw hex dump of test_ns.dcf (ns with x=42, name='hello', vec=⍳10):
- Type_code = 0x0000 (unique namespace marker, both type and rank bytes zero)
- data_words field = 8 (small relative to ~2KB payload — internal structure has own length)
- Offset +0x12 (18): value 4 (constant across all NS variants?)
- Offset +0x1a (26): signature `00 14 0E 14` — related to DCF format version marker 0x14
- Member names stored in UTF-16LE (found 'x' at +0xC8, 'vec' at +0xE0, 'name' at +0xF8 in test_ns.dcf)
- Name descriptors: pattern `01 28 00 88 00 00 00 00` where 0x28 = Char16 type byte
- Member values use standard 220⌶ payload format with size_words prefix
  - 'hello' at +0x110: data_words=5, type=0x271F (Char8 vector), count=5, "hello"
  - ⍳10 at +0x130: data_words=6, type=0x221F (Int8 vector), count=10, [0..9]
  - 42 at +0x158: data_words=4, type=0x220F (Int8 scalar), value=42
- Large character/symbol table at +0x300..+0x800 (~1.3KB) containing ASCII, APL symbols as u32 LE
- Each member adds exactly 48 bytes overhead
- Names ≤4 chars fit in an 8-byte slot; 'longname' (8 chars, 16 UTF-16 bytes) adds 16 extra bytes
- The table at +0x284..+0x2F6 contains lowercase a-z as u32 LE values (0x61-0x7A), then digits, then uppercase A-Z — this is the APL atomic vector / character translation table

## Running Dyalog Scripts
```
D:\devel\dyalog\20.0\dyascript.exe "APLKEYS=D:\devel\dyalog\20.0\aplkeys" "APLTRANS=D:\devel\dyalog\20.0\apltrans" -script "script.apls"
```
- Run from DCFFiles directory
- .apls files need UTF-8 BOM (﻿ at start)
- **dyascript does NOT support ∇ (nabla) function definitions** — all code must be inline
- Use `⎕FCREATE⍠('J' 0)('C' 1)('Z' 0)` variant for file properties (NOT `⎕FPROPS`)
- Use `:Trap 0 ... :Else ... :EndTrap` for error handling
- `⎕NDELETE file` can fail on non-existent files — wrap in `:Trap`
- `⎕OR` fails in dyascript mode (DOMAIN ERROR)

## Complete 220⌶ Type Byte Mapping (VERIFIED)
| Byte | ⎕DR | Type | Size |
|------|------|------|------|
| 0x21 | 11 | Boolean | 1 bit MSB-first |
| 0x22 | 83 | Int8 | 1 byte |
| 0x23 | 163 | Int16 | 2 bytes LE |
| 0x24 | 323 | Int32 | 4 bytes LE |
| 0x25 | 645 | Float64 | 8 bytes LE |
| 0x06 | 326 | Nested/Pointer | variable recursive |
| 0x27 | 80 | Char8 | 1 byte |
| 0x28 | 160 | Char16 | 2 bytes LE |
| 0x29 | 320 | Char32 | 4 bytes LE |
| 0x2A | 1289 | Complex128 | 16 bytes |
| 0x2E | 1287 | Decimal128 | 16 bytes BID |
| 0x00 | — | Namespace | variable ~2KB+ |

No Int64 type exists — integers ≥2³¹ promote to Float64.
</technical_details>

<important_files>
- `probe_ns_deep.apls`
  - Creates 13 controlled NS variants and dumps all serialized bytes
  - Key for namespace format reverse-engineering
  - Tests: empty, 1/2/3 int members, long name, float, string, nested, sub-NS, 5-mixed, different name lengths

- `analyze_ns.py`
  - Python binary analyzer for namespace serialization
  - Parses DUMP output, finds UTF-16 strings and type codes, does byte-level diffs
  - **Currently being used to analyze the output — analysis was interrupted**

- `ns_probe_output.txt`
  - Raw output with all 13 namespace byte dumps as decimal values
  - Input for analyze_ns.py; contains the complete serialized bytes for each variant

- `ns_analysis_output.txt`
  - Output from analyze_ns.py (~291KB)
  - Contains size analysis, detailed structure analysis, and byte-level diffs
  - **Needs to be read and interpreted to complete the namespace format mapping**

- `probe_decimal128.apls`
  - Confirmed Decimal128 type byte = 0x2E via ⎕FR←1287 + arithmetic
  - DCF creation had file path issue but serialization data was captured in console output

- `test_ns_variants.dcf`
  - DCF file with 13 namespace components for binary analysis
  - Can be analyzed with analyze_serialization.py or direct hex inspection

- `analyze_serialization.py`
  - Fixed in this session: matrix shapes, nested rank detection, namespace type, Complex128/Char32/Bool display
  - Now correctly parses all 57+ test components

- `docs/component-blocks.md`
  - Major update with complete type mapping, rank patterns, shape encoding, nested format, namespace notes
  - **Still needs**: Decimal128 type byte correction (currently says 0x2B hypothesized, should say 0x2E confirmed), full namespace internal structure

- `docs/README.md`
  - Updated: moved array serialization to "Fully Documented", added test files, added surprising findings

- `C:\Users\stf\.copilot\session-state\658af01e-f952-45f1-bc79-5d896d7c456b\plan.md`
  - Updated plan with Phase A (Decimal128) and Phase B (Namespace deep-dive)
</important_files>

<next_steps>
Immediate next steps (in progress when compacted):
1. **Read and interpret `ns_analysis_output.txt`** — the analysis output has size comparisons, UTF-16 string locations, type code locations, and byte-level diffs between all key pairs. Need to study the diff output to map the namespace internal structure.

2. **Map the namespace internal structure** from the differential analysis:
   - Header/metadata region (first ~0xA0 bytes, appears constant across all NS)
   - Member name table: where names are stored, how they're indexed, the `01 28 00 88` descriptor pattern
   - Member value table: size_words prefix + standard 220⌶ payload
   - Character/symbol translation table (~1.3KB at +0x300..+0x800)
   - Any index/pointer structures linking names to values
   - The ordering relationship (names appear in one order, values in another)

3. **Update docs with Decimal128 correction** — change type byte from "0x2B hypothesized" to "0x2E confirmed" in component-blocks.md

4. **Write comprehensive namespace format documentation** — update docs/component-blocks.md with the mapped internal structure

5. **Consider additional experiments** if the differential analysis leaves gaps:
   - NS with many members (10+) to see if the 48-byte-per-member pattern holds
   - NS with very long names to understand name storage scaling
   - NS with functions (⎕FX) to understand function serialization within NS

Open questions:
- What exactly is in the ~0xA0 bytes of NS header before member data?
- What is the `01 28 00 88` descriptor pattern for member names? (0x28 = Char16 type, 0x88 = flags?)
- Why do member names and values appear in different orders?
- What is the full purpose of the ~1.3KB character table? (APL atomic vector? Translation table?)
- The field at +0x12 (value=4) — what does it represent?
- The signature `00 14 0E 14` at +0x1a — is this always the same?
</next_steps>