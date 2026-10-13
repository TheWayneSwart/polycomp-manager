# Polycomp Manager (Windows app)

A single-file Windows program for sending text to PolyComp AMP2008 LED signs (192x32, 4 rows)
over a serial cable. Runs on Windows 7+ and under Wine. No installer, no runtime needed.

## Using it

1. Plug the USB-to-RS232 adapter into the PC and the null modem cable into the sign (the sign's
   internal GSM modem must be unplugged, see ../CLAUDE.md).
2. Start `PolycompManager.exe`. Pick the serial port (Device Manager shows the COM number of the
   adapter; the newest one is usually the highest number). Leave the address at 0.
3. Choose a layout: 4 small rows, a big top row + 2 small, 2 small + a big bottom row, or 2 big rows.
   Small rows hold 32 characters, big rows 16. "Wide" characters count double.
4. Type each row, pick colour / flash / wide / alignment. The preview shows what the board will do.
5. "Send to sign". The sign answers with an acknowledgement; the status line confirms it.
   The page is stored in the sign and survives power-off.
6. "Add page" makes a sequence: pages are shown one after another for the chosen number of
   seconds. The effect "Scroll bottom row (slide)" keeps the upper rows static and scrolls row 4;
   its time setting is then the scroll speed.
7. "Save message..." / "Open message..." keep messages as small `.json` files (see `../examples/`).
   The last message and port are restored on the next start.
8. "Set sign clock" sends this PC's date and time to the sign.

The same executable can be scripted:

    PolycompManager.exe --send message.json --port COM3
    PolycompManager.exe --set-clock --port COM3
    PolycompManager.exe --send message.json --dry-run      (prints the frames as hex)

## Building (on Linux)

    ./build.sh            # needs Go in ~/.local/go/bin or on PATH; produces dist/PolycompManager.exe (+ 32-bit)

`go test ./...` runs the protocol, layout and model tests. The GUI (`cmd/polycomp-gui`) only
compiles for `GOOS=windows`; the protocol packages compile everywhere.

## Testing under Wine on Linux

    ln -s /dev/ttyUSB0 ~/.wine/dosdevices/com33      # once; pick a free comN
    sg dialout -c "wine dist/PolycompManager.exe"       # or log in again after joining dialout

Wine's background server opens device files on the program's behalf, so it must itself run with
the `dialout` group: if a Wine process was started before you joined the group, run `wineserver -k`
first. The Windows build uses plain synchronous Win32 serial calls precisely because Wine mishandles
the overlapped reads most serial libraries use.
