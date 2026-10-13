# Probe for the STM32 factory USART bootloader. Sends 0x7F (autobaud sync); bootloader
# answers 0x79 (ACK). Then asks for the chip product ID (command 0x02). Read-only.
import serial, time
for baud in (9600, 115200, 57600, 38400, 19200):
    s = serial.Serial("/dev/ttyUSB0", baud, parity="E", timeout=0.5)
    s.reset_input_buffer()
    s.write(b"\x7f"); got = s.read(4)
    print(f"baud={baud:6d} 8E1 sync 7F -> {got.hex(' ') if got else '(nothing)'}", flush=True)
    if got and got[-1] == 0x79:
        s.write(b"\x02\xfd"); pid = s.read(8)
        print(f"  GET_ID reply: {pid.hex(' ')}", flush=True)
        s.write(b"\x01\xfe"); ver = s.read(8)
        print(f"  GET_VERSION reply: {ver.hex(' ')}", flush=True)
        s.close(); break
    s.close()
