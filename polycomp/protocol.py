"""GTXC frame encoding. Pure functions, no serial I/O.

Reference: polycomp.co.za/protocol.html as archived in Feb 2008 (link in docs/README.md), corrected
against the AMP2008 v02.22 firmware (see CLAUDE.md "Firmware findings"): never set SERST B1
(interrupt), the firmware drops such frames; Nol must be 0 or the configured line count; the
vendor's PAGE STATUS 0xC0 (B6 background) floods the board green, use 0x80.
"""
from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime
from enum import IntEnum

SYNC = 0x00
ETX = 0x03
EOT = 0x04
ACK = 0x06
NAK = 0x15
CTRL = 0x1C  # inline formatting prefix

ENCODING = "cp437"  # "extended IBM ASCII"


class Effect(IntEnum):
    AUTO = 0x0
    APPEAR = 0x1
    WIPE = 0x2
    OPEN = 0x3
    LOCK = 0x4
    ROTATE = 0x5
    RIGHT = 0x6
    LEFT = 0x7
    ROLL_UP = 0x8
    ROLL_DOWN = 0x9
    PING_PONG = 0xA
    FILL_UP = 0xB
    PAINT = 0xC
    FADE_IN = 0xD
    JUMP = 0xE
    SLIDE = 0xF


# TEMPO index -> seconds on screen (or scroll speed in SLIDE mode, 1 fastest)
TEMPO_SECONDS = {1: 2, 2: 5, 3: 10, 4: 20, 5: 30, 6: 45, 7: 60, 8: 90, 9: 120}


class DisplayMode(IntEnum):
    TIMER = 0b00       # follow schedule
    ALWAYS_ON = 0b10   # B5 set (vendor example 0xE2)
    ALWAYS_OFF = 0b01  # B4 set


def serst(*, interrupt=False, more_pages=False, ack=False, schedule=False) -> int:
    v = 0xC0
    if interrupt:
        v |= 1 << 1
    if more_pages:
        v |= 1 << 2
    if ack:
        v |= 1 << 3
    if schedule:
        v |= 1 << 4
    return v


def tempo(index: int = 2, mode: DisplayMode = DisplayMode.ALWAYS_ON) -> int:
    if index not in TEMPO_SECONDS:
        raise ValueError(f"tempo index must be 1-9, got {index}")
    return 0xC0 | (int(mode) << 4) | index


def function(effect: Effect = Effect.APPEAR, *, clock=False, temperature=False) -> int:
    v = 0xC0 | int(effect)
    if clock:
        v |= 1 << 4
    if temperature:
        v |= 1 << 5
    return v


def page_status(*, bold_pairs: int = 0, auto_centre=False, foreign=False, background=False) -> int:
    """bold_pairs: bitmask B0..B3 joining line pairs (1+2, 3+4, 5+6, 7+8)."""
    v = 0x80 | (bold_pairs & 0x0F)
    if auto_centre:
        v |= 1 << 4
    if foreign:
        v |= 1 << 5
    if background:
        v |= 1 << 6
    return v


def checksum(data: bytes) -> int:
    ck = 0
    for b in data:
        ck ^= b
    return ck


def fmt(cmd: str) -> bytes:
    """Inline control block: F flash, E enlarge, R/G/Y/M/W colour, D default (W seen in firmware)."""
    if cmd not in "FERGYMWD" or len(cmd) != 1:
        raise ValueError(f"unknown control command {cmd!r}")
    return bytes([CTRL]) + cmd.encode("ascii")


@dataclass
class Page:
    text: str
    number: int = 1
    tempo: int = 0xE2
    function: int = 0xC1
    status: int = 0xC0
    schedule: bytes = b""  # 17 bytes when SERST schedule bit set, else empty

    def body(self) -> bytes:
        if self.schedule and len(self.schedule) != 17:
            raise ValueError("schedule block must be exactly 17 bytes")
        return (
            self.schedule
            + bytes([self.tempo, self.function, self.status])
            + self.text.encode(ENCODING)
        )


def clock_page(now: datetime | None = None) -> Page:
    """Page 000: HHMMSSDDMM0W, W = day of week with 1 = Sunday."""
    now = now or datetime.now()
    dow = (now.isoweekday() % 7) + 1  # Mon=1..Sun=7 -> Sun=1..Sat=7
    text = now.strftime("%H%M%S%d%m") + f"0{dow}"
    p = Page(text=text, number=0)
    return p


def build_frame(page: Page, *, lines: int, address: int = 0, flags: int = 0xC0) -> bytes:
    if not 0 <= address <= 127:
        raise ValueError("address must be 0-127")
    if not 0 <= page.number <= 999:
        raise ValueError("page number must be 0-999")
    if bool(flags & 0x10) != bool(page.schedule):
        raise ValueError("schedule bit in flags must match presence of schedule block")
    head = bytes([SYNC, lines, address, ETX, flags]) + f"{page.number:03d}".encode("ascii")
    if page.number == 0:
        body = page.text.encode("ascii")  # clock page has no tempo/function/status
    else:
        body = page.body()
    frame = head + body + bytes([EOT])
    return frame + bytes([checksum(frame)])


def hexdump(data: bytes) -> str:
    return " ".join(f"{b:02X}" for b in data)
