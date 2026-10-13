"""Send hand-built frames and print any reply.
Usage: probe_frames.py "LABEL:HEXBODY" ...   (checksum is appended; e.g. "S form:00 53 00 03 48 49 04")
"""
import os, sys, time, serial
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from polycomp.protocol import checksum, hexdump

with serial.Serial("/dev/ttyUSB0", 9600, timeout=0.2) as s:
    for arg in sys.argv[1:]:
        name, _, hexbody = arg.rpartition(":")
        body = bytes.fromhex(hexbody); f = body + bytes([checksum(body)])
        s.reset_input_buffer()
        print(f"{time.strftime('%H:%M:%S')} TX [{name}] {hexdump(f)}", flush=True)
        s.write(f); s.flush()
        got = b""; end = time.monotonic() + 3
        while time.monotonic() < end: got += s.read(64)
        print(f"         RX {hexdump(got) if got else '(nothing)'}", flush=True)
        time.sleep(20)
