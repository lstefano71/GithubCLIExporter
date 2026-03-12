<overview>
The user is reverse-engineering the Dyalog APL Component File (DCF) binary format, specifically focusing on how arrays of all ⎕DR data types are serialized within components using the 220⌶ (Serialise Array) i-beam. The project already has extensive knowledge of the DCF file structure (header, directory, journal, checksums, compression, free blocks) but the array serialization format was only partially mapped (5 of 11+ types). My approach was to create APL probe scripts to serialize every type, run them via dyascript.exe, create DCF test files, and analyze the binary output with Python.
</overview>

<history>
1. User asked to create a plan for reverse-engineering array serialization format in DCF components, pointing to ⎕DR documentation showing more types than previously mapped, and requesting namespace serialization experiments.
   - Fetched ⎕DR docs (https://docs.dyalog.com/20.0/language-reference-guide/system-functions/data-representation-monadic/) listing 11 types: Bool(11), Char8(80), Int8(83), Char16(160), Int16(163), Char32(320), Int32(323), Pointer(326), Float64(645), Dec128(1287), Complex128(1289)
   - Fetched 220⌶ docs (https://docs.dyalog.com/20.0/language-reference-guide/primitive-operators/i-beam/serialise-deserialise-array/) - key example: `'ab' → DF A4 05 00...`
   - Explored codebase thoroughly via explore agents to understand current state
   - Created plan.md and SQL todos

2. User said to actually RUN the scripts, not just create them. Pointed out the interpreter is at D:\devel\dyalog\20.0\ and I should figure out execution again.
   - Used explore agent to find execution pattern in existing Python scripts
   - Command: `D:\devel\dyalog\20.0\dyascript.exe "APLKEYS=D:\devel\dyalog\20.0\aplkeys" "APLTRANS=D:\devel\dyalog\20.0\apltrans" -script "script.apls"`
   - Ran probe_serialize_types.apls - GOT COMPLETE TYPE MAPPING (all types confirmed!)
   - Ran probe_serialize_nested.apls - GOT NESTED ARRAY FORMAT
   - Ran probe_serialize_ns.apls (first run had ⎕FPROPS syntax error, fixed and re-ran)
   - Fixed ⎕FPROPS syntax: changed from `⎕FPROPS tn 'J' 0` to `⎕FCREATE⍠('J' 0)('C' 1)('Z' 0)` variant
   - Ran create_test_types.apls - created 5 DCF test files successfully
   - Ran analyze_serialization.py - confirmed type mappings in actual DCF binary files
</history>

<work_done>
Files created:
- `probe_serialize_types.apls`: Comprehensive APL script serializing every ⎕DR type at multiple ranks. EXECUTED SUCCESSFULLY.
- `probe_serialize_nested.apls`: Nested array serialization deep-dive (fixed typo ⎎→⎕). EXECUTED SUCCESSFULLY.
- `probe_serialize_ns.apls`: Namespace serialization experiments (fixed ⎕FPROPS→⎕FCREATE⍠ syntax). EXECUTED SUCCESSFULLY.
- `create_test_types.apls`: Creates 5 DCF files with all types (fixed ⎕FPROPS→⎕FCREATE⍠ syntax). EXECUTED SUCCESSFULLY.
- `analyze_serialization.py`: Python parser for DCF component serialization. EXECUTED SUCCESSFULLY (but has bugs in matrix shape parsing - see issues below).

DCF test files generated:
- `test_types_basic.dcf` (25 components: bool/int8/int16/int32/float64 scalars+vectors+matrices+boundary cases)
- `test_types_char.dcf` (11 components: char8/char16/char32 scalars+vectors+matrices)
- `test_types_complex.dcf` (5 components: complex128 scalar+vector+matrix, decimal128 scalar+vector)
- `test_types_nested.dcf` (8 components: various nested arrays)
- `test_types_rank.dcf` (8 components: rank 1-4 arrays)
- `test_ns.dcf` (1 component: namespace with x=42, name='hello', vec=⍳10)

Work completed:
- [x] Created and ran type probing script - COMPLETE TYPE BYTE MAPPING DISCOVERED
- [x] Created and ran nested array probe - NESTED FORMAT FULLY DOCUMENTED
- [x] Created and ran namespace probe - NS SERIALIZATION WORKS (huge discovery!)
- [x] Created and ran DCF test file creator - all 5 files generated
- [x] Created and ran Python analyzer - confirmed mappings in binary
- [ ] Python analyzer has bugs in matrix/higher-rank shape parsing (reads wrong offsets)
- [ ] Documentation not yet updated with findings
- [ ] analyze_serialization.py needs fixes for nested type rank decoding and matrix shapes

Known issues with analyze_serialization.py:
- Matrix shape parsing reads wrong offsets - treats data as shape dimensions (e.g. shape=[3, 66186119091210] is garbage)
- The DCF block layout has a backup header between primary header and payload that the parser doesn't account for (the component block in C=1 files has extra backup block fields)
- Nested arrays show rank=-1 because the rank byte pattern (0x07 low nibble) isn't handled
- Namespace type (0x00) isn't recognized
</work_done>

<technical_details>
## COMPLETE 220⌶ SERIALIZATION FORMAT

### Preamble (10 bytes):
- Bytes 0-1: `0xDF 0xA4` (magic)
- Bytes 2-5: `u32` size_words = (total_bytes - 2) / 8 + 1
- Bytes 6-9: `00 00 00 00` (reserved/padding)

### Type/Rank Word (8 bytes at offset 10):
- Byte 10: rank_byte
- Byte 11: type_byte
- Bytes 12-17: zeros

### Rank Byte Encoding:
- Simple arrays: `(rank << 4) | 0x0F` → 0x0F=scalar, 0x1F=vector, 0x2F=matrix, 0x3F=rank3, 0x4F=rank4
- Nested/pointer arrays: `(rank << 4) | 0x07` → 0x07=nested scalar, 0x17=nested vector, 0x27=nested matrix

### CONFIRMED Type Byte Mapping:
| Type Byte | Decimal | ⎕DR | Type | Element Size |
|-----------|---------|------|------|-------------|
| 0x21 | 33 | 11 | Boolean | 1 bit (packed MSB-first) |
| 0x22 | 34 | 83 | Int8 | 1 byte signed |
| 0x23 | 35 | 163 | Int16 | 2 bytes signed LE |
| 0x24 | 36 | 323 | Int32 | 4 bytes signed LE |
| 0x25 | 37 | 645 | Float64 | 8 bytes IEEE 754 LE |
| 0x06 | 6 | 326 | Nested/Pointer | variable (recursive) |
| 0x27 | 39 | 80 | Char8 | 1 byte |
| 0x28 | 40 | 160 | Char16 | 2 bytes UTF-16 LE |
| 0x29 | 41 | 320 | Char32 | 4 bytes UTF-32 LE |
| 0x2A | 42 | 1289 | Complex128 | 16 bytes (real64+imag64) |
| 0x00 | 0 | 326 | Namespace | special format (~2KB min) |

NOTE: 0x2B for Decimal128 was hypothesized but NOT observed. ⎕FR←1287 still produced Float64 (type 0x25). Decimal128 may need a special Dyalog build.

### Data Layout After Type/Rank Word:
- **Scalars (rank 0)**: Value directly in next 8 bytes (or 16 for complex)
- **Vectors (rank 1)**: u64 element_count, then data padded to 8 bytes
- **Matrices (rank 2)**: u64 dim1, u64 dim2, then data padded to 8 bytes
- **Rank N**: N × u64 shape dimensions, then data padded to 8 bytes

### Type Promotion Boundaries (integers):
- 0..127, ¯128..¯1 → Int8 (⎕DR 83)
- 128..32767, ¯32768..¯129 → Int16 (⎕DR 163)
- 32768..2147483647, ¯32769..below → Int32 (⎕DR 323)
- ≥2147483648 → **Float64** (⎕DR 645) — NO Int64 TYPE EXISTS!

### Boolean Bit Packing:
Bits packed MSB-first: `1 0 1 0 1 1 0 0` → byte 0xAC (10101100)

### Nested Array Format:
- Type byte = 0x06, rank byte low nibble = 0x07
- After outer header (type/rank + element_count for vectors, + shape dims for matrices):
- Each sub-element: u64 size_words + recursive serialization (same format minus DF A4 preamble)
- size_words matches what would be in bytes 2-5 of standalone 220⌶ of that element
- Deep nesting is fully recursive (nested vectors of nested vectors work)
- `⊂scalar` where scalar is simple → simplifies to just the scalar (APL rule)
- `⊂vector` → nested scalar (rank_byte=0x07) containing the vector

### Namespace Serialization:
- 220⌶ WORKS directly on namespaces (⎕NC=9, ⎕DR=326)
- Empty NS serializes to ~1986 bytes (contains interpreter metadata/symbol table)
- NS with members: ~2154 bytes (members add ~168 bytes for 3 simple members)
- Nested NS works: ~2394 bytes
- NS with functions: ~2474 bytes
- Type/rank word at offset 10 is ALL ZEROS (0x00 0x00...) — unique namespace marker
- NS round-trips through DCF files: write with ⎕FAPPEND, read back with ⎕FREAD, member access works
- ⎕OR fails in dyascript mode (DOMAIN ERROR)

### DCF Component Block Layout (C=1):
The Python analyzer has issues because the DCF block format for C=1 files includes:
- Primary block header (24 bytes at block start)
- Payload at +0x18: data_words(8) + type_code(8) + element_count(8) + data...
- A backup header also exists (64 bytes) interleaved somewhere
- The `alloc_size` field at block+0x00 covers the entire component including backup

### Running Dyalog Scripts:
```
D:\devel\dyalog\20.0\dyascript.exe "APLKEYS=D:\devel\dyalog\20.0\aplkeys" "APLTRANS=D:\devel\dyalog\20.0\apltrans" -script "script.apls"
```
- Run from the DCFFiles directory
- .apls files need UTF-8 BOM (﻿ at start)
- Use `⎕FCREATE⍠('J' 0)('C' 1)('Z' 0)` variant for setting file properties at creation (NOT `⎕FPROPS tn 'J' 0` which causes SYNTAX ERROR in dyascript)
- Use `:Trap 0 ... :Else ... :EndTrap` for error handling
- `⎕NDELETE file` to delete files (not `⎕FERASE file ⎕FTIE 0`)
</technical_details>

<important_files>
- `probe_serialize_types.apls`
   - Primary experiment script - serializes EVERY ⎕DR type at every rank
   - Created new, executed successfully, output contains the complete type byte mapping
   - Key sections: Phase 1 (⎕DR verification), Phase 2 (220⌶ for all types), Phase 3 (higher rank shapes)

- `probe_serialize_nested.apls`
   - Deep-dive into nested/pointer arrays (⎕DR 326)
   - Created new, fixed typo (⎎→⎕ on line ~72), executed successfully
   - Confirmed: type_byte=0x06, rank low nibble=0x07, recursive sub-array format

- `probe_serialize_ns.apls`
   - Namespace serialization experiments
   - Created new, fixed ⎕FPROPS syntax (lines ~126-128, ~157-159), executed successfully
   - Key finding: 220⌶ works on NS directly, NS round-trips through DCF, ⎕OR fails in dyascript

- `create_test_types.apls`
   - Creates 5 DCF test files with all data types
   - Created new, fixed ⎕FPROPS→⎕FCREATE⍠ syntax in CreateDCF helper function
   - Generated: test_types_basic/char/complex/nested/rank.dcf

- `analyze_serialization.py`
   - Python parser for DCF component serialization format
   - Created new, has BUGS in matrix shape parsing (reads element_count field at wrong offset for C=1 files)
   - Needs fix: nested array rank decoding (low nibble 0x07), matrix shape offset, namespace type

- `docs/` directory
   - Contains existing DCF format documentation (not yet updated with new findings)

- `create_tests.apls`
   - Reference for correct APL syntax (⎕FCREATE⍠ variant, ⎕NDELETE)

- `C:\Users\stf\.copilot\session-state\658af01e-f952-45f1-bc79-5d896d7c456b\plan.md`
   - Session plan file with structured investigation phases
</important_files>

<next_steps>
Remaining work:
1. **Fix analyze_serialization.py** - Matrix/higher-rank shape parsing is broken (reads wrong offsets because C=1 backup block header isn't accounted for). Need to understand the exact component block layout for C=1 files.
2. **Fix nested array rank decoding** in analyzer - rank_byte low nibble 0x07 needs to be handled (currently shows rank=-1)
3. **Add namespace type (0x00)** recognition to analyzer
4. **Update documentation** in docs/ with complete type byte mapping, nested format, namespace findings
5. **Write a "how to run dyascript" note** - user requested this for future reference
6. **Deeper namespace format analysis** - the ~2KB internal structure of namespace serialization is complex and mostly unexplored (contains what appears to be symbol tables, metadata)

Immediate priorities:
- Fix the Python analyzer to correctly parse all component types
- Document the complete findings (the type mapping, nested format, NS format)
- The SQL todos table has 'update-docs' as the only pending todo

Open questions:
- Why does Decimal128 (⎕DR 1287) not appear even with ⎕FR←1287? May need specific Dyalog build
- What is the exact internal structure of namespace serialization? (2KB of complex metadata)
- Is there an Int64 type hidden somewhere, or does APL truly promote to Float64 at 2^31?
- The backup block structure in C=1 files needs to be properly accounted for in the analyzer
</next_steps>