# Read-only STM32 bootloader memory reader (command 0x11 only). Usage: stm32read.py ADDR LENGTH OUTFILE
import serial, struct, sys, time
addr = int(sys.argv[1], 0); length = int(sys.argv[2], 0); out = sys.argv[3]
s = serial.Serial("/dev/ttyUSB0", 9600, parity="E", timeout=3)
def expect_ack(what):
    b = s.read(1)
    if b != b"\x79":
        raise SystemExit(f"{what}: expected ACK got {b.hex() if b else 'nothing'}")
def read_block(a, n):
    s.write(b"\x11\xee"); expect_ack("READ cmd")
    pkt = struct.pack(">I", a); s.write(pkt + bytes([pkt[0]^pkt[1]^pkt[2]^pkt[3]])); expect_ack("address")
    s.write(bytes([n-1, (n-1)^0xff])); expect_ack("length")
    data = s.read(n)
    if len(data) != n: raise SystemExit(f"short read at {a:#x}: {len(data)}")
    return data
t0 = time.time()
with open(out, "wb") as f:
    done = 0
    while done < length:
        n = min(256, length-done)
        f.write(read_block(addr+done, n)); done += n
        if done % 16384 == 0 or done == length:
            print(f"{done}/{length} bytes, {time.time()-t0:.0f}s", flush=True)
print("done")
