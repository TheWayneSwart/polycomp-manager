"""Row layout for the 192x32 AMP2008 sign (4 rows of 8 px, 32 characters per row).

Verified live 2026-10-12: the text field is a plain stream that fills rows in order, wrapping
every 32 characters. CR, LF, RS, US and ESC do not break lines (ESC + 3 bytes starts a new
*segment* that is shown after the current one). A row pair joined via PAGE STATUS B0 (rows 1+2)
or B1 (rows 3+4) still consumes 64 characters of text but renders only the first 16, tall.
So "line breaks" are made by padding each line to the width its row consumes.

Inline markup: {F} flash, {E} enlarge, {R} red, {G} green, {Y} yellow, {M} multicolour,
{W} white, {D} default. These become 0x1C control blocks and take no room on the row.
"""
from __future__ import annotations

import re

ROW_CHARS = 32          # characters on one 8 px row
TALL_VISIBLE = 16       # characters actually shown on a joined (16 px) row pair
PAIR_CHARS = 2 * ROW_CHARS  # text consumed by a joined pair
ROWS = 4

CTRL = 0x1C
MARKUP = re.compile(r"\{([FERGYMWD])\}")


def expand_markup(line: str) -> tuple[str, int]:
    """Replace {X} markup with 0x1C control blocks; return (text, cells_used).

    {E} (enlarge) doubles character width until {D}, so those characters count as 2 cells.
    """
    cells, wide, pos = 0, False, 0
    for m in MARKUP.finditer(line):
        cells += (m.start() - pos) * (2 if wide else 1)
        pos = m.end()
        if m.group(1) == "E":
            wide = True
        elif m.group(1) == "D":
            wide = False
    cells += (len(line) - pos) * (2 if wide else 1)
    text = MARKUP.sub(lambda m: chr(CTRL) + m.group(1), line)
    return text, cells


def layout(lines: list[str], *, tall_top: bool = False, tall_bottom: bool = False,
           centre: bool = False, align: str | None = None) -> tuple[str, int]:
    """Lay out `lines` one per visual row. Returns (page_text, page_status_bold_bits).

    Visual rows, top to bottom: pair 1+2 is one tall row if tall_top else two normal rows;
    likewise pair 3+4 with tall_bottom. align is "left" (default), "centre" or "right" and is
    done with leading spaces (not PAGE STATUS B4, whose behaviour with padding is unverified);
    centre=True is shorthand for align="centre".
    """
    align = align or ("centre" if centre else "left")
    if align not in ("left", "centre", "right"):
        raise ValueError(f"align must be left, centre or right, not {align!r}")
    rows: list[tuple[int, int]] = []  # (visible width, consumed width)
    for tall in (tall_top, tall_bottom):
        rows += [(TALL_VISIBLE, PAIR_CHARS)] if tall else [(ROW_CHARS, ROW_CHARS)] * 2
    if len(lines) > len(rows):
        raise ValueError(f"{len(lines)} lines but only {len(rows)} rows with this join setting")
    out = []
    for i, line in enumerate(lines):
        text, visible = expand_markup(line)
        width, consumed = rows[i]
        if visible > width:
            raise ValueError(f"line {i + 1} is {visible} characters, row holds {width}: {line!r}")
        lead = {"left": 0, "centre": (width - visible) // 2, "right": width - visible}[align]
        last = i == len(lines) - 1
        if last:
            out.append(" " * lead + text)
        else:
            # reset formatting first so padding spaces are single width and the next row starts clean
            reset = chr(CTRL) + "D" if MARKUP.search(line) else ""
            out.append(" " * lead + text + reset + " " * (consumed - visible - lead))
    bold_bits = (0b01 if tall_top else 0) | (0b10 if tall_bottom else 0)
    return "".join(out), bold_bits
