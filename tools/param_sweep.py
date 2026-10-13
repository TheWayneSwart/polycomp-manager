import serial, sys, time
sys.path.insert(0, __import__("os").path.dirname(__import__("os").path.dirname(__import__("os").path.abspath(__file__))))
from polycomp.protocol import Page, build_frame, serst, hexdump

def try_send(ser, lines, address):
    frame = build_frame(Page(text="HELLO WAYNE", number=1), lines=lines, address=address,
                        flags=serst(interrupt=True, ack=True))
    ser.reset_input_buffer()
    ser.write(frame); ser.flush()
    got = ser.read(8)
    return got

hits = []
# Phase 1: 9600 8N1, every address, Nol 4
with serial.Serial("/dev/ttyUSB0", 9600, timeout=0.7) as ser:
    for addr in range(0, 128):
        got = try_send(ser, 4, addr)
        if got:
            print(f"HIT addr={addr} -> {hexdump(got)}", flush=True); hits.append(("9600 8N1", 4, addr, got))
print("phase 1 (addresses) done", flush=True)

# Phase 2: address 0 and 1, every baud/parity/stop/Nol combination
for baud in (1200, 2400, 4800, 9600, 19200, 38400):
    for parity in ("N", "E", "O"):
        for stop in (1, 2):
            with serial.Serial("/dev/ttyUSB0", baud, parity=parity, stopbits=stop, timeout=0.7) as ser:
                for lines in (1, 2, 4):
                    for addr in (0, 1):
                        got = try_send(ser, lines, addr)
                        if got:
                            print(f"HIT baud={baud} {parity}{stop} lines={lines} addr={addr} -> {hexdump(got)}", flush=True)
                            hits.append((f"{baud} 8{parity}{stop}", lines, addr, got))
print("phase 2 (serial params) done", flush=True)
print("TOTAL HITS:", len(hits))
