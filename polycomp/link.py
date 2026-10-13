"""Serial transport for GTXC frames."""
from __future__ import annotations

import logging
import time

import serial

from .protocol import ACK, NAK, hexdump

log = logging.getLogger(__name__)


class SignLink:
    def __init__(self, port: str, baud: int = 9600, stopbits: int = 1, timeout: float = 3.0):
        self.ser = serial.Serial(
            port, baud, bytesize=serial.EIGHTBITS, parity=serial.PARITY_NONE,
            stopbits=serial.STOPBITS_TWO if stopbits == 2 else serial.STOPBITS_ONE,
            timeout=timeout,
        )

    def close(self):
        self.ser.close()

    def send(self, frame: bytes, *, expect_ack: bool, timeout: float = 3.0) -> bytes:
        log.info("TX %d bytes: %s", len(frame), hexdump(frame))
        self.ser.reset_input_buffer()
        self.ser.write(frame)
        self.ser.flush()
        if not expect_ack:
            return b""
        deadline = time.monotonic() + timeout
        got = b""
        while time.monotonic() < deadline:
            chunk = self.ser.read(1)
            if chunk:
                got += chunk
                if chunk[0] in (ACK, NAK):
                    break
        log.info("RX %d bytes: %s", len(got), hexdump(got))
        if not got:
            log.warning("no reply from sign")
        elif got[-1] == NAK:
            log.warning("sign replied NAK")
        return got

    def drain(self, seconds: float) -> bytes:
        """Read whatever the sign sends for a while (useful for sniffing self-test chatter)."""
        self.ser.timeout = 0.2
        end = time.monotonic() + seconds
        got = b""
        while time.monotonic() < end:
            got += self.ser.read(256)
        if got:
            log.info("RX %d bytes: %s", len(got), hexdump(got))
        return got
