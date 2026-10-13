import serial, sys, time
sys.path.insert(0, __import__("os").path.dirname(__import__("os").path.dirname(__import__("os").path.abspath(__file__))))
from polycomp.protocol import Page, build_frame, serst, hexdump

frame = build_frame(Page(text="HELLO WAYNE", number=1), lines=4, address=0,
                    flags=serst(interrupt=True, ack=True))
ser = serial.Serial("/dev/ttyUSB0", 9600, timeout=0.1)
ser.reset_input_buffer()
t0 = time.monotonic(); end = t0 + float(sys.argv[1])
def log(msg): print(f"{time.strftime('%H:%M:%S')} +{time.monotonic()-t0:6.1f}s {msg}", flush=True)
log("responder running, power-cycle the sign now")
while time.monotonic() < end:
    got = ser.read(64)
    if not got: continue
    log(f"RX {hexdump(got)}  {got!r}")
    if 0x05 in got:
        ser.write(b"\x06"); ser.flush(); log("TX 06 (ACK) in reply to ENQ")
        time.sleep(0.2)
        ser.write(frame); ser.flush(); log(f"TX test page {hexdump(frame)}")
        reply = ser.read(16)
        log(f"RX after page: {hexdump(reply) if reply else '(nothing)'}")
log("responder finished")
