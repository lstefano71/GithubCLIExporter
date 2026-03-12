<overview>
The user is reverse-engineering the Dyalog APL Component File (DCF) binary format, specifically the internal array serialization used by `220⌶` (Serialise Array). This session focused on completing the namespace serialization analysis: fixing the sub-array stream parser to correctly decode the internal structure of serialized namespaces, confirming the advance formula for size_words-prefixed sub-arrays, and mapping the complete three-section layout (names → values → internal metadata). The approach is experimental: create APL scripts that serialize controlled test values via `dyascript.exe`, then analyze the binary output with Python differential analysis.
</overview>

<history>
1. Session resumed from prior checkpoint where namespace binary analysis was in progress
   - Read `ns_analysis_output.txt` to find key diff patterns between NS variants
   - Discovered that same-size diffs (NS_A1 vs NS_AB1, NS_ABC1, NS_AFLOAT, NS_X42) showed precisely where member names and values are stored
   - Name at P+0xC0: UTF-16LE, only the name bytes differ between variants
   - Value at P+0xE0: only the type_code or value bytes differ

2. Created `analyze_ns_deep.py` for comprehensive structural analysis
   - Full hex dumps with u64 annotations, type code detection, UTF-16 string detection
   - Character table location finder (a-z in u32 LE)
   - Ran successfully, saved output to `ns_deep_output.txt`

3. Identified the sub-array stream structure starting at P+0x0098
   - Each namespace payload contains a stream of size_words-prefixed sub-arrays
   - Initial parsing showed names in reverse alphabetical order, then values, then metadata
   - Created `parse_ns_stream.py` for stream parsing

4. Debugged the advance formula for the sub-array stream
   - First attempt: `pos += (sw-1)*8` (advance only data, not sw word) — appeared to work for inner entries
   - Changed to `pos += sw*8` — BROKE parsing (jumped too far, read descriptors as size_words)
   - Systematic verification: tested both advance formulas at each position in NS_A1
   - **CONFIRMED: advance = (sw-1)*8, data = (sw-2)*8 bytes**
   - For sw=4: block is 24 bytes total (8 sw + 16 data), advance 24
   - For sw=7: block is 48 bytes total (8 sw + 40 data), advance 48
   - TERM (sw=0): advance 8

5. Fixed `parse_ns_stream.py` and ran complete analysis of all 13 variants
   - Successfully parsed all variants except NS_CHILD (which has embedded child NS with different marker)
   - Confirmed the three-section structure: NAMES → TERM → VALUES → TERM → INTERNAL_METADATA → 5×TERM

6. Verified header and metadata consistency across all 13 variants
   - Header (P+0x00..0x98): **identical across ALL 13 variants** — 152 bytes of fixed workspace metadata
   - Internal metadata (last 1752 bytes): **identical across ALL variants except NS_CHILD**
   - NS_CHILD has 728 differing metadata bytes (embeds child NS with its own metadata copy)

7. Was viewing `docs/component-blocks.md` to update with findings when compaction was triggered
</history>

<work_done>
Files created:
- `analyze_ns_deep.py`: Comprehensive namespace binary analysis tool — hex dumps, type detection, UTF-16 finder, character table locator. Output in `ns_deep_output.txt`.
- `parse_ns_stream.py`: Sub-array stream parser for namespace payloads. Correctly parses all 13 variants with section identification (names/values/metadata).

Files from prior checkpoint (still relevant):
- `probe_ns_deep.apls`: Creates 13 controlled NS variants and dumps serialized bytes
- `ns_probe_output.txt`: Raw output with all 13 namespace byte dumps
- `ns_analysis_output.txt`: Output from `analyze_ns.py` (~291KB)
- `test_ns_variants.dcf`: DCF file with 13 namespace components

Work completed:
- [x] Analyze ns_analysis_output.txt — extracted key diff patterns
- [x] Create comprehensive analysis tool (analyze_ns_deep.py)
- [x] Debug and confirm sub-array stream advance formula
- [x] Fix parse_ns_stream.py with correct formula
- [x] Successfully parse all 13 namespace variants
- [x] Verify header consistency (identical across all variants)
- [x] Verify metadata consistency (identical except NS_CHILD)
- [x] Map complete three-section namespace structure
- [ ] **IN PROGRESS**: Update docs/component-blocks.md with namespace findings
- [ ] **IN PROGRESS**: Update Decimal128 type byte from 0x2B to 0x2E in docs
- [ ] Document namespace format in detail

SQL todos status:
- decimal128: done
- ns-probe: done
- ns-analyze: in_progress (analysis complete, documentation pending)
- ns-docs: pending (depends on ns-analyze)
</work_done>

<technical_details>
## Sub-Array Stream Format (CRITICAL FINDING)
The namespace payload (and nested array values) use a stream of size_words-prefixed sub-arrays:
```
[sw:u64] [data: (sw-2)*8 bytes]
```
- **advance** = `(sw-1) * 8` bytes from current position
- **data** = `(sw-2) * 8` bytes, starting at `pos + 8`
- **TERM** = sw=0, advance 8 bytes
- sw=1 is invalid (data would be negative)
- sw=4: 16 bytes data (2 words), 24 bytes total block
- sw=7: 40 bytes data (5 words), 48 bytes total block

This same format applies to nested array sub-elements AND namespace member entries.

## Complete Namespace Serialization Structure

### Preamble (bytes 0-9, before payload)
- Bytes 0-1: `DF A4` magic
- Bytes 2-5: u32 LE `size_words`
- Bytes 6-9: zeros

### Payload Layout
```
P+0x0000: type_code = 0x0000000000000000 (namespace marker)
P+0x0008: field_8 = 4 (constant)
P+0x0010: signature = 00 14 0E 14 00 00 00 00 (constant)
P+0x0018..0x0097: Fixed header (152 bytes, identical across all NS variants)
                   Contains workspace markers: 0x50D5, 0x5505, 0x58D5
                   Contains value 0x72 (='r'), value 14, value 8
P+0x0098: Start of sub-array stream
```

### Sub-Array Stream Sections
1. **NS_SELF_NAME**: `[sw=4] [01 00 00 88 ...] [FF FF 00 00 ...]` — always anonymous (0xFFFF)
2. **MEMBER_NAMES** (reverse-sorted alphabetically):
   - Each: `[sw=4+] [01 28 00 88 ...] [name in UTF-16LE ...]`
   - sw=4 for names ≤ 4 chars (16 bytes fits: 8-byte descriptor + 8-byte name slot)
   - sw=6 for "longname" (8 chars = 16 UTF-16 bytes → needs extra 16 bytes)
3. **TERM** (sw=0)
4. **MEMBER_VALUES** (same order as names, i.e., reverse-sorted):
   - Each is a standard 220⌶ sub-array: `[sw] [type_code] [count/value] [data...]`
   - Int8 scalar: sw=4, Float64 scalar: sw=4, Char8 "hello": sw=5, Int8[3]: sw=5, Bool[3]: sw=5
5. **TERM** (sw=0)
6. **INTERNAL METADATA** (~1752 bytes, identical across all simple NS variants):
   - Three `0x50D5` marker blocks (sw=7 each, 40 bytes data)
   - Type0x15 scalar with constant value `0x9b2ba1869b84063d` (possibly workspace hash/ID)
   - Various Int8 and Type0x14 scalars (workspace state)
   - Char8[0] empty string
   - Type0x14[256] + Int8[256] — character translation table (the APL atomic vector)
   - Type 0x1E scalar with value 1
   - Type0x14 scalar with value 0x285
   - Five TERM entries (sw=0) to end the stream

### Name Descriptor Format
```
01 28 00 88  — member name descriptor
01 00 00 88  — namespace self-name descriptor (followed by 0xFFFF for anonymous)
01 98 00 88  — child namespace member descriptor (seen in NS_CHILD)
```
Pattern: `01 TT 00 88` where TT indicates the member type:
- 0x00 = namespace self-name
- 0x28 = Char16 (regular variable name)
- 0x98 = child namespace reference

### Per-Member Overhead
Each member adds exactly 48 bytes for simple scalar values:
- 24 bytes for name entry (sw=4: 8-byte sw + 16-byte descriptor+name)
- 24 bytes for value entry (sw=4: 8-byte sw + 16-byte type+value)
Larger values (strings, vectors) add more in the value section.

### NS_CHILD Special Case
- Child namespace member uses descriptor `01 98 00 88` instead of `01 28 00 88`
- The child NS value embeds a complete namespace payload inline
- Uses marker `0x5505` instead of `0x0617` (Nested) in the D550 wrapper block
- Parser breaks at sw=1 inside the embedded NS (different internal structure)
- The metadata section has 728 bytes different from simple NS (contains child's own metadata copy)

## Decimal128 (⎕DR 1287) — Confirmed from prior checkpoint
- **Type byte = 0x2E** (decimal 46), NOT 0x2B as docs currently say
- Activation: `⎕FR←1287` BEFORE computation. Literals are parsed before ⎕FR takes effect.
- Element size: 16 bytes IEEE 754-2008 BID format
- BID: sign(1 bit) + exponent(14 bits, biased 6176) + coefficient(113 bits)

## Running Dyalog Scripts
```
D:\devel\dyalog\20.0\dyascript.exe "APLKEYS=D:\devel\dyalog\20.0\aplkeys" "APLTRANS=D:\devel\dyalog\20.0\apltrans" -script "script.apls"
```
- .apls files need UTF-8 BOM
- dyascript does NOT support ∇ function definitions — all code must be inline
- `⎕OR` fails in dyascript mode (DOMAIN ERROR)
- Use `:Trap 0 ... :Else ... :EndTrap` for error handling

## Type Byte Mapping (VERIFIED, needs doc update)
| Byte | ⎕DR | Type | Size |
|------|------|------|------|
| 0x21 | 11 | Boolean | 1 bit MSB-first |
| 0x22 | 83 | Int8 | 1 byte |
| 0x23 | 163 | Int16 | 2 bytes LE |
| 0x24 | 323 | Int32 | 4 bytes LE |
| 0x25 | 645 | Float64 | 8 bytes LE |
| 0x06 | 326 | Nested | variable recursive |
| 0x27 | 80 | Char8 | 1 byte |
| 0x28 | 160 | Char16 | 2 bytes LE |
| 0x29 | 320 | Char32 | 4 bytes LE |
| 0x2A | 1289 | Complex128 | 16 bytes |
| **0x2E** | **1287** | **Decimal128** | **16 bytes BID** |
| 0x00 | — | Namespace | variable ~2KB+ |

Additional type codes found inside namespace metadata:
- 0x14: Unknown internal type (used for workspace state values)
- 0x15: Unknown internal type (workspace hash/ID?)
- 0x1E: Unknown internal type (single scalar, value=1)
- 0x50D5, 0x5505, 0x58D5: Workspace structure markers (NOT type codes — raw u16 values in D550 header blocks)
</technical_details>

<important_files>
- `parse_ns_stream.py`
  - THE key analysis tool — correctly parses namespace sub-array streams
  - Fixed advance formula: `(sw-1)*8`, data = `(sw-2)*8` bytes
  - Identifies three sections: names, values, internal metadata
  - Full output saved to temp file showing all 13 variants parsed correctly

- `docs/component-blocks.md`
  - Main documentation file that needs updating with namespace findings
  - Lines 289-311: Current namespace section — needs major rewrite with the decoded structure
  - Line 166: Decimal128 type byte says 0x2B — needs correction to 0x2E
  - Line 171: Says Decimal128 "not observed" — needs correction to "confirmed"
  - Line 427: Data types table says 0x2B hypothesised — needs correction

- `analyze_ns_deep.py`
  - Comprehensive binary analysis tool with hex dump, type detection, character table finder
  - Output in `ns_deep_output.txt`

- `probe_ns_deep.apls`
  - APL script creating 13 controlled NS variants for differential analysis
  - All variants: EMPTY_NS, NS_A1, NS_AB, NS_ABC, NS_AB1, NS_ABC1, NS_X42, NS_LONGNAME, NS_AFLOAT, NS_ASTR, NS_ANESTED, NS_CHILD, NS_5MIXED

- `ns_probe_output.txt`
  - Raw dump output from probe_ns_deep.apls — input for all analysis scripts

- `test_ns_variants.dcf`
  - DCF file with all 13 namespace components — can be analyzed with other tools

- `docs/README.md`
  - Updated in prior checkpoint — may need further updates after namespace docs are complete
</important_files>

<next_steps>
Remaining work:
1. **Update `docs/component-blocks.md`** with complete namespace serialization findings:
   - Replace lines 289-311 (current partial NS section) with full decoded structure
   - Fix Decimal128 type byte: 0x2B → 0x2E (lines 166, 171, 427)
   - Add sub-array stream format documentation (advance formula, data length)
   - Document the name descriptor types (0x00, 0x28, 0x98)
   - Document internal metadata section (character table, workspace state)
   - Note NS_CHILD special case

2. **Update SQL todos**: Mark ns-analyze as done, start ns-docs

3. **Optional further experiments**:
   - NS with many members (10+) to verify 48-byte pattern holds
   - NS with functions (⎕FX) — though ∇ not available in dyascript
   - Better understanding of Type0x14, Type0x15, Type0x1E in metadata
   - Decode the 0x50D5 marker block internal structure

Immediate next action:
- Edit `docs/component-blocks.md` to add complete namespace format documentation and fix Decimal128 type byte
</next_steps>