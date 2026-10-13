"""polycomp send [--port /dev/ttyUSB0] [--lines 4] [--address 0] "TEXT"

Example:  python -m polycomp.cli send --dry-run "HELLO"
"""
from __future__ import annotations

import argparse
import logging
import sys
import time

from . import protocol as P


def main(argv=None):
    ap = argparse.ArgumentParser(prog="polycomp")
    sub = ap.add_subparsers(dest="cmd", required=True)

    s = sub.add_parser("send", help="send one or more pages of text")
    s.add_argument("text", nargs="+",
                   help="one argument per page; separate rows with a newline or literal \\n; "
                        "inline markup {R} {G} {Y} {F} {E} {D} (see polycomp/layout.py)")
    s.add_argument("--port", default="/dev/ttyUSB0")
    s.add_argument("--baud", type=int, default=9600)
    s.add_argument("--stopbits", type=int, default=1, choices=[1, 2])
    s.add_argument("--lines", type=int, default=4, help="Nol byte: text lines on the sign")
    s.add_argument("--address", type=int, default=0, help="sign address, 0 = all-call")
    s.add_argument("--effect", default="APPEAR", choices=[e.name for e in P.Effect])
    s.add_argument("--tempo", type=int, default=2, help="1-9, see TEMPO_SECONDS")
    s.add_argument("--align", choices=["left", "centre", "right"], default="left",
                   help="align each row by padding on the PC side (default left)")
    s.add_argument("--centre", action="store_true", help="same as --align centre")
    s.add_argument("--auto-centre-bit", action="store_true", help="set PAGE STATUS B4 (sign-side auto-centre, unverified)")
    s.add_argument("--background", action="store_true",
                   help="set PAGE STATUS B6 (fills the board green with red bars between characters)")
    s.add_argument("--clock", action="store_true", help="show clock on top line")
    s.add_argument("--interrupt", action="store_true",
                   help="interrupt mode (not stored). WARNING: AMP2008 firmware v02.22 silently drops these frames")
    s.add_argument("--no-ack", action="store_true", help="do not request ACK/NAK after each page (default: request it)")
    s.add_argument("--set-time", action="store_true", help="send page 000 with current time first")
    s.add_argument("--tall", choices=["top", "bottom", "both"],
                   help="join row pairs into 16 px rows: rows 1+2, 3+4 or both (16 chars per tall row)")
    s.add_argument("--dry-run", action="store_true", help="print frames, do not open the port")

    l = sub.add_parser("listen", help="dump whatever the sign sends for N seconds")
    l.add_argument("--port", default="/dev/ttyUSB0")
    l.add_argument("--baud", type=int, default=9600)
    l.add_argument("--seconds", type=float, default=10)

    a = ap.parse_args(argv)
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")

    if a.cmd == "listen":
        from .link import SignLink
        link = SignLink(a.port, a.baud)
        try:
            link.drain(a.seconds)
        finally:
            link.close()
        return 0

    from .layout import layout
    pages = []
    if a.set_time:
        pages.append(P.clock_page())
    for i, t in enumerate(a.text, start=1):
        lines = t.replace("\\n", "\n").split("\n")
        text, bold = layout(lines, tall_top=a.tall in ("top", "both"), tall_bottom=a.tall in ("bottom", "both"),
                            align="centre" if a.centre else a.align)
        pages.append(P.Page(
            text=text,
            number=i,
            tempo=P.tempo(a.tempo),
            function=P.function(P.Effect[a.effect], clock=a.clock),
            status=P.page_status(bold_pairs=bold, auto_centre=a.auto_centre_bit, background=a.background),
        ))

    frames = []
    for idx, pg in enumerate(pages):
        last = idx == len(pages) - 1
        flags = P.serst(interrupt=a.interrupt, more_pages=not last, ack=not a.no_ack)
        frames.append(P.build_frame(pg, lines=a.lines, address=a.address, flags=flags))

    if a.dry_run:
        for f in frames:
            print(P.hexdump(f))
        return 0

    from .link import SignLink
    link = SignLink(a.port, a.baud, stopbits=a.stopbits)
    try:
        for idx, f in enumerate(frames):
            link.send(f, expect_ack=not a.no_ack, timeout=3.0 if idx == 0 else 0.5)
            if a.no_ack:
                time.sleep(0.5)
    finally:
        link.close()
    return 0


if __name__ == "__main__":
    sys.exit(main())
