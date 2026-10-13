# CLAUDE.md — polycomp

Keep this file current. Update the **Status log** and **Open questions** sections whenever
something is learned, decided or built. Treat it as the project's working memory.

## What this project is

Replacement controller software for two PolyComp LED text display boards bought second-hand
from a client. The vendor's original software no longer exists. Goal: a small app that pushes
new text to the boards over a serial link using PolyComp's **GTXC protocol**.

## Hardware

- **Boards:** 2 x PolyComp, label inside reads `Model: RGT32/P10-4L 192x32 HB`.
  P10 LED modules, 192 x 32 pixels. "4L" is probably 4 text lines of 8 px (unverified; could
  also be run as 2 lines of 16 px). The protocol's `Nol` byte must match what the sign expects.
- **"Modem" from the client:** a box with an RS232-to-USB converter, a GSM-style antenna and a
  SIM slot. One board has a matching modem inside with a SIM number sticker. Working theory:
  the original setup updated the signs remotely over GSM (SMS or CSD data call), with the
  internal modem feeding the sign's serial board.
- **Plan:** bypass GSM entirely. Connect the USB-serial converter straight to the sign's serial
  communication board (3 wires: TXD, RXD, GND, crossed over).
- **Cabling:** the sign's serial port is a male DB9, same as the USB adapter, so the supplied
  male-to-female cable cannot connect them. A DB9 female-to-female **null modem** cable is the
  right one (verified live 2026-10-12: data flows both ways through it).
- The serial board has **no DIP switches** (earlier assumption was wrong). On power-up the sign
  self-tests and displays its serial settings on the screen (shows 9600).
- **Serial board: PolyComp AMP2008 Ver 4.11 (MAR 2014)**, photos in `images/`. STM32F103 high-density MCU (bootloader PID 0x0414,
  bootloader v2.2, 512 KB flash, no read protection), DS3231 RTC, RESET and BOOT buttons, **no DIP
  switches**. Serial is a 5-way header marked `USART1`: blue in position 1, 2 empty, green in 3,
  red in 4, 5 empty, with a **MAX3232** level shifter beside it (proper RS232). Also on the board:
  large SOIC at U9 (probably SPI flash: pages + fonts; no SD card fitted, microSD footprint empty),
  3.6 V Ni-MH backup cell for the RTC, `IR_Sensor` header (on-screen editor is driven by an IR
  remote, not a keyboard), unpopulated USART2 and two TTL USART headers, two display outputs.
  That cable ends in a male DB9 which plugged straight into the internal GSM
  modem (modem = DCE, so the board end is DTE-wired). Power-up self-test screen shows `9600`.
- **Working chain (verified both directions):** board USART1 cable (male DB9) -> DB9 F/F null modem
  cable -> FTDI FT232R USB adapter (`/dev/ttyUSB0`, idVendor 0403:6001, in-kernel `ftdi_sio`).
  Loopback through adapter + null modem cable passes; STM32 bootloader answers through the full
  chain. **The internal GSM modem must be unplugged**; with it in the chain nothing gets through.
- **Boot behaviour:** ~1-2 s after power-up the board transmits ENQ (`0x05`) three times, 1 s apart,
  at 9600 8N1. Replying ACK (`0x06`) and/or a GTXC frame in that window changes nothing.
- **STM32 bootloader access:** hold BOOT, press+release RESET, release BOOT. Then `0x7F` at 9600
  8E1 gets `0x79`. Scratch scripts used: `stm32probe.py` (sync/GET_ID/GET_VERSION) and
  `stm32read.py` (READ MEMORY only). Never send ERASE (0x43/0x44) or WRITE (0x31). Plain RESET
  returns to the sign firmware. Full flash dumped to `firmware/` (git-ignored: vendor code plus the
  previous owner's messages and phone numbers).
- Dev machine: Linux, Python 3.14, pyserial 3.5 installed system-wide. **No pip, venv or pytest**
  on this machine (python3-venv / python3-pytest not installed; needs sudo apt). Run tests with
  `python3 run_tests.py` until pytest is available. Check `dmesg | tail` and `lsusb` after plugging
  the adapter in; user must be in the `dialout` group (added 2026-10-12; a shell started before
  that needs `sg dialout -c "..."` or a re-login).

## GTXC protocol summary

Source: https://web.archive.org/web/20080218105448/http://www.polycomp.co.za/protocol.html (local git-ignored copy may sit at `docs/gtxc-protocol-2008-archive.html`). This section is the working summary; consult the archive for the full tables.

**Serial:** 9600 baud, 8 data bits, 1 stop bit, no parity (defaults, DIP-switch changeable).
The vendor's BASIC sample opened the port with 2 stop bits; 1 is the documented default. Character
set is "extended IBM ASCII" (likely CP437).

**Frame** (one page per frame, pages sent one after another):

| Field | Bytes | Value |
|---|---|---|
| Sync | 1 | `0x00` |
| Nol | 1 | number of text lines on the sign (binary, not ASCII) |
| Address | 1 | 1–127, `0` = all-call |
| ETX | 1 | `0x03` end of header |
| SERST | 1 | serial status flags, see below |
| Page number | 3 | ASCII `"000"`–`"999"` |
| Schedule block | 17 | **only present when SERST B4 = 1**, otherwise omitted entirely |
| TEMPO | 1 | see below |
| FUNCTION | 1 | see below |
| PAGE STATUS | 1 | see below |
| Text | n | ASCII message, with optional `0x1C` control blocks |
| EOT | 1 | `0x04` |
| Checksum | 1 | XOR of every byte from Sync through EOT inclusive |

**SERST bits:** B0 = 0, B5 = 0, B6 = B7 = 1.
B1 interrupt mode (single page, not stored, shown until next message, lost on power-off).
B2 more pages to come (clear on last page; sign times out after 40 s and shows what it has).
B3 ACK requested (sign replies `0x06` ACK or `0x15` NAK after each page).
B4 schedule mode (schedule block present).
Useful values: `0xC0` plain, `0xC8` plain + ACK, `0xCC` more pages + ACK.

**Page 000** is the clock: ASCII `HH MM SS DD MM 0W` (12 chars), W = day of week, 1 = Sunday.
Optional; only send to set the RTC.

**TEMPO:** B0–B3 duration index 1–9 = 2, 5, 10, 20, 30, 45, 60, 90, 120 s. In Slide mode it is
the scroll speed instead (1 fastest, 9 slowest). B4/B5 display mode: both 0 = timer (per
schedule), B5 = 1 = always on, B4 = 1 = always off. B6 = B7 = 1. Vendor example uses `0xE2`
(always on, 5 s).

**FUNCTION:** B0–B3 effect: 0 auto, 1 appear, 2 wipe, 3 open, 4 lock, 5 rotate, 6 right, 7 left,
8 roll up, 9 roll down, A ping-pong, B fill up, C paint, D fade in, E jump, F slide.
B4 clock on top line, B5 temperature on top line, B6 = B7 = 1. Vendor example uses `0xC1` (appear).

**PAGE STATUS:** B0–B3 join line pairs (1+2, 3+4, 5+6, 7+8) as bold/double height, B4 auto-centre,
B5 foreign language, B6 background colour on, B7 = 1. Vendor example uses `0xC0`.

**Inline formatting:** `0x1C` followed by one letter: `F` flash, `E` enlarge, `R` red, `G` green,
`Y` yellow, `M` multicolour, `D` back to default. Every page starts as red, normal size, no flash.

**Timing:** allow up to 3 s for the first page's ACK and 0.5 s for each page after that.

**Reference frame** from the vendor's BASIC sample (1-line sign, address 2, page 001):

```
00 01 02 03 C0 30 30 31 E2 C1 C0  <ASCII text>  04  <xor>
```

Use this as the first unit-test fixture for the frame builder.

## Firmware findings (from the flash dump, 2026-10-12)

Firmware string `Firmware v02.22`, `Generic RGBT`, `PolyComp`. USART1 RX ISR -> byte state
machine at `0x08017238` (state byte `0x200001ee`, XOR accumulator `0x200001e0`, RX buffer from
`0x2000b483`). Disassemble with `tools/fwdis.py ADDR LEN` (needs capstone on PYTHONPATH; a wheel is
unpacked in the session scratchpad, or `sudo apt install python3-capstone`).

**Verified live against the sign (ACK = `0x06` received):**

- Long form works **only without SERST B1 (interrupt)**: `00 Nol Addr 03 C8 "001" E2 C1 C0 text 04 xor`
  -> ACK. The same frame with `0xCA` (interrupt + ACK) is silently dropped. `0xC0` (no ACK bit)
  is accepted silently (no reply, by design).
- `Nol` may be `0` (accepted unconditionally) or the configured line count. **`Nol = 4` is
  accepted**, so the sign is configured as 4 lines. `Nol = 'S'` selects the short form.
- Address byte: `0` (all-call) or configured address + 1 (firmware stores it zero-based).
- **Short form** (not in the 2008 doc): `00 'S' Addr 03 text 04 xor`. Firmware substitutes header
  `C8 "001" E1 C1 80` itself, i.e. it writes stored page 001 and ACKs.
- Checksum = XOR of every byte from Sync through EOT, as documented.
- SERST `?` or `S` sets a status flag; `R` sets another flag (untested, may clear pages). A SERST
  with `(SERST & 0xC1) == 0x80` selects a **bitmap upload** path: 3-digit id, 3-digit width,
  3-digit height, then width*height/8 bytes (x2 if B4, x3 if B5), EOT, xor.
- Boot ENQ x3 is sent by the sign, not expected from us; ignoring it is fine.
- **Row layout (verified live):** the text field is a plain stream filling 4 rows of 32
  characters in order, wrapping every 32. CR, LF, RS (0x1E), US (0x1F) do **not** break lines.
  ESC + TEMPO + FUNCTION + PAGE STATUS inside the text starts a new *segment* shown after the
  current one (the firmware's own demo page uses this). PAGE STATUS B0 joins rows 1+2 and B1 rows
  3+4 into one 16 px row each; a joined pair still **consumes 64 characters** of text and renders
  the first 16 tall. Line breaks are therefore made by padding: `polycomp/layout.py` does this
  (`{R} {G} {Y} {F} {E} {W} {D}` markup -> 0x1C codes, zero width). Text beyond the rows is
  shown as further screens at the TEMPO interval.
- **SLIDE effect verified:** FUNCTION effect F keeps the upper rows static and scrolls **row 4**
  right-to-left; TEMPO index is then the speed (1 fastest, 9 slowest; 3 looks right). Row 4 text
  may be any length. `0x1C E` (enlarge) = **double width**, same height, so a 19-char enlarged
  line overflows a static row but is fine on the sliding row. Recipe for a static heading plus
  marquee: `send --tall top --centre --effect SLIDE --tempo 3 'HEADING\n\n{Y}{E}scrolling text'`.
- **Hardware fault on this board:** the green LEDs on the two leftmost P10 modules are dead (red
  works there). Yellow text turns red on those modules. User plans to replace the modules.
- **Font:** CP437 graphics in the control range do not render (`0x11` left-pointer shows nothing).
  Use ASCII for arrows, e.g. `<--`. Lowercase and `&` render fine.
- **CLI gotcha:** each positional argument is a *page*; one page with several rows must be a single
  argument with `\n` between rows. Extra arguments create pages 002+, which the sign stores.
- **Colours and flash verified:** `0x1C G` green, `0x1C Y` yellow, `0x1C F` flash all work.
  Firmware also knows `W` (seen in its demo page; probably white on RGB boards).
- **Display verified:** stored page 001 shows on the sign. The sign is red/green (tri-colour with
  yellow). PAGE STATUS B6 "background" (vendor sample `0xC0`) fills the whole board green with red
  bars between characters; `0x80` is the sane default. CLI now defaults to ACK on, background off,
  and warns about `--interrupt`.

## Open questions / unverified

- **Why SERST B1 (interrupt mode) makes the firmware drop the frame** (it passes the receive
  state machine, so the main loop must reject it). Find the main-loop consumer of the RX buffer at
  `0x2000b483` and the "frame complete" flag at `0x20000001`.
- What the `?` status query (SERST = `0x3F`, sets flag `0x20000055`) and the `R` variant (sets
  `0x200002dd`, possibly "rub"/clear) actually do; `?` produced no serial reply.
- Where stored pages live: no SD card on the board, flash regions outside the app are blank, so
  probably an external SPI/I2C memory. Photo of the board pending.
- Meaning of the three boot-time ENQs (controller detection? modem probe?).
- `Nol` resolved: 4 (0 also accepted). Line separation resolved: fixed-width rows, see Hardware.
- Sign address: no DIP switches, so it lives in firmware config. Unknown.
- Whether `0x1C W` renders (white on RGB; maybe yellow or nothing here).
- Multi-page sequences (`more pages` bit, pages 002+) and whether the old stored pages 002+ still
  exist and would resume cycling after ours; also how to delete pages (`R` SERST variant? the
  on-screen editor's `RUB PAGE`/`CLEAR ALL` via IR remote).
- The second board has not been tested yet; expect the same modem-bypass procedure.
- Why the sign showed only the clock after one power-cycle (with the modem still attached) but the
  stored speed-limit pages after later ones. Possibly a schedule window on those pages.

## Stack and conventions

- **Python 3 + pyserial.** CLI first, a web or GUI front end later if wanted.
- Keep the protocol encoding in a pure module with no serial I/O, so it can be unit-tested
  against the reference frame above without hardware.
- Planned layout:
  - `polycomp/protocol.py` — frame building, checksum, enums for TEMPO / FUNCTION / PAGE STATUS.
  - `polycomp/link.py` — pyserial transport, ACK/NAK handling, retries, timeouts.
  - `polycomp/cli.py` — e.g. `polycomp send --port /dev/ttyUSB0 --address 0 "HELLO"`.
  - `tests/` — pytest.
- Every send path must have a `--dry-run` / hex-dump option so frames can be inspected without
  hardware.
- Log raw bytes in both directions when talking to the sign; the protocol is undocumented
  beyond the 2008 page and we will be reverse-engineering details.
- Never commit SIM numbers, IMEIs or anything from the stickers inside the boards.
- Not yet a git repository (`.gitignore` is in place for when it is).
- Run: `python3 -m polycomp.cli send --dry-run "HELLO"` prints frames as hex without a port.
  `python3 -m polycomp.cli listen --seconds 10` dumps anything the sign transmits.
  Real send, rows separated by `\n`, tall top row, colours:
  `python3 -m polycomp.cli send --tall top 'HELLO WAYNE\n{G}AMP2008 LINK OK\n{Y}ROW FOUR'`
  (defaults: 9600 8N1, Nol 4, address 0, ACK requested, background off, TEMPO 5 s, APPEAR).
  `--align left|centre|right` aligns each row by PC-side padding (`--centre` = `--align centre`);
  `--auto-centre-bit` sets PAGE STATUS B4 instead (sign-side, unverified). Lowercase is sent
  as-is (cp437). `--tall both` gives two 16 px rows of 16 characters each.
- `tools/`: `probe_frames.py 'label:hex'` sends raw frames and prints replies; `fwdis.py` disassembles
  the dump; `stm32_probe.py` / `stm32_read.py` talk to the STM32 bootloader (read-only);
  `loopback.py`, `enq_responder.py`, `param_sweep.py` were the diagnostics used on 2026-10-12.

## Windows app (`winapp/`)

Go module producing `winapp/dist/PolycompManager.exe` (64-bit) and `PolycompManager-32bit.exe`, built
on Linux with `winapp/build.sh` (Go lives in `~/.local/go`, user-local install, no sudo). GUI is
`lxn/walk` (pure Go Win32, needs the embedded Common-Controls manifest, see `build.sh`). Packages:
`internal/gtxc` (protocol + layout, same fixtures as the Python tests), `internal/link` (serial;
Windows backend is hand-written synchronous Win32 because Wine breaks overlapped reads),
`cmd/polycomp-gui` (window, preview, settings in `%APPDATA%\PolycompManager`, headless `--send`
mode). Verified 2026-10-12: the exe running under Wine 10 sent a page through `COM33` ->
`/dev/ttyUSB0` and got the sign's ACK; GUI screenshot-checked via Xephyr (`tools`-free recipe in
`winapp/README.md`). Example message in `examples/sample-menu.json`. Not yet run on real Windows.

## Status log

- **2026-10-08** Project started. Protocol doc recovered from the Wayback Machine and archived in
  `docs/`.
- **2026-10-08** First code: `polycomp/protocol.py` (frame builder, flag helpers, clock page),
  `polycomp/link.py` (pyserial transport with ACK/NAK wait and a sniff/drain helper),
  `polycomp/cli.py` (`send` and `listen` subcommands, `--dry-run`), `tests/test_protocol.py`
  (6 tests, all passing, including the vendor BASIC sample frame byte-for-byte). CLI defaults match
  the vendor sample: TEMPO 0xE2, FUNCTION 0xC1, PAGE STATUS 0xC0, Nol 4, address 0, CR between
  lines. Nothing sent to real hardware yet. Next: plug in the adapter, `listen` during the sign's
  power-up self-test, then `send` the reference frame.
- **2026-10-08** Blocked on hardware: need a DB9 female-to-female null modem cable before the
  first live test. Chosen: Communica XY-PC39 (DB9 F/F null modem, ~R114, Samrand branch).
  Rejected along the way: plain gender changers and "extension" cables (straight-through),
  Digi-Key AIM 30-9506-77 (extension). Existing USB-serial adapter stays in use.
- **2026-10-12** First live session. Cable arrived; adapter is an FTDI FT232R, works out of the box.
  Sign powers up fine and still shows the previous owner's stored pages (speed-limit notices,
  control-room phone number) plus a correct clock. With the internal GSM modem in the chain
  nothing passed in either direction, including `AT` probes (so the DB9 was the modem's port).
  Unplugged the modem and connected the board's USART1 cable to the adapter via the null modem
  cable: the board's boot-time ENQ x3 now arrives cleanly at 9600 8N1, loopback passes, and the
  STM32 factory bootloader answers (STM32F103 high-density, 512 KB, unprotected). So the link is
  good, but the sign firmware ignores every GTXC variant tried. Dumped the full flash to
  `firmware/stm32f103-512k-dump.bin` for reverse engineering. Scratch tools (sweep, responder,
  loopback, bootloader probe/reader) live in the session scratchpad; promote the useful ones into
  the repo next.
- **2026-10-12 (later)** Firmware reverse-engineered far enough to talk to the sign. Root cause of
  the silence: the firmware drops any frame with SERST B1 (interrupt mode) set, and every earlier
  test used it. Long-form frames with SERST `0xC8` and `Nol` 0 or 4 now get ACK, as does the
  undocumented short `'S'` form. `tools/probe_frames.py` sends ad-hoc frames. Next: confirm on the
  screen what the accepted frames displayed, fix CLI defaults (no interrupt mode), find the page
  storage, decode TEMPO/FUNCTION handling in the main loop.
- **2026-10-12 (end of session)** Sign fully usable from the CLI. Confirmed live: ACK on every
  accepted frame; page 001 displays; colours and flash work; rows are 32 chars, joined pairs
  consume 64 chars and show 16 tall; CR is not a separator. Added `polycomp/layout.py` (padding
  layout + `{X}` markup), CLI `--tall top|bottom|both`, defaults changed (ACK on, background
  off, `--interrupt` warned), 7 layout tests (13 total, all passing). Last page sent through the
  CLI at 11:59: tall `HELLO WAYNE`, green `AMP2008 LINK OK`, yellow `ROW FOUR`, ACKed.
  Still not a git repository.
- **2026-10-12 (afternoon)** First real content on the sign: tall centred heading plus a marquee
  row (SLIDE, enlarge, colours). Found two P10 modules with dead green on the left of the board.
  Added `--centre` (PC-side padding) and `--auto-centre-bit`; 14 tests passing.
- **2026-10-12 (afternoon, later)** Built the Windows app in Go (`winapp/`): GUI with pages list,
  per-row editors, board preview, send / set clock / save / load, headless mode; cross-compiled
  to 64- and 32-bit exes; proven end-to-end under Wine against the real sign. Also fixed a Python
  layout bug (padding after a wide row was double width; now resets with `0x1C D` first).
