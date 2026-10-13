"""Disassemble a region of the firmware dump. Usage: dis.py ADDR [NBYTES]  (needs capstone on PYTHONPATH)"""
import sys, struct, capstone
d = open("firmware/stm32f103-512k-dump.bin", "rb").read(); BASE = 0x08000000
md = capstone.Cs(capstone.CS_ARCH_ARM, capstone.CS_MODE_THUMB | capstone.CS_MODE_MCLASS); md.detail = True
addr = int(sys.argv[1], 0) & ~1; n = int(sys.argv[2], 0) if len(sys.argv) > 2 else 0x200
for i in md.disasm(d[addr-BASE:addr-BASE+n], addr):
    note = ""
    if i.mnemonic.startswith("ldr") and "[pc" in i.op_str:
        imm = i.operands[1].mem.disp; pool = ((i.address + 4) & ~3) + imm
        v = struct.unpack_from("<I", d, pool-BASE)[0]; note = f"; ={v:#010x}"
        if 0x08000000 <= v < 0x08080000:
            s = d[v-BASE:v-BASE+32].split(b"\0")[0]
            if s and all(32 <= c < 127 for c in s): note += f' "{s.decode()}"'
    print(f"{i.address:#x}  {i.mnemonic:8s} {i.op_str:34s}{note}")
