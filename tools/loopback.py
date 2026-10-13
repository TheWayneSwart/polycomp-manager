import serial, sys
s = serial.Serial("/dev/ttyUSB0", 9600, timeout=1)
s.reset_input_buffer()
s.write(b"LOOPBACK-TEST-123")
got = s.read(32)
print("sent  LOOPBACK-TEST-123")
print("got  ", got)
print("RESULT:", "LOOPBACK OK, adapter and cable TX/RX path fine" if got == b"LOOPBACK-TEST-123" else "NO LOOPBACK")
