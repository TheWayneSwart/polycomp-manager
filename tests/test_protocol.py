from polycomp import protocol as P


def test_vendor_single_line_example():
    # From the 2008 BASIC sample: CHR$(0)+CHR$(1)+CHR$(2)+CHR$(3)+CHR$(&HC0)+"001"
    #   +CHR$(&HE2)+CHR$(&HC1)+CHR$(&HC0) + "CNT = 100" + CHR$(4) + xor
    page = P.Page(text="CNT = 100", number=1)
    frame = P.build_frame(page, lines=1, address=2, flags=0xC0)
    expected_head = bytes([0x00, 0x01, 0x02, 0x03, 0xC0]) + b"001" + bytes([0xE2, 0xC1, 0xC0])
    assert frame.startswith(expected_head)
    assert frame[len(expected_head):-2] == b"CNT = 100"
    assert frame[-2] == 0x04
    assert frame[-1] == P.checksum(frame[:-1])


def test_vendor_double_line_example_all_call():
    page = P.Page(text="X", number=1)
    frame = P.build_frame(page, lines=2, address=0)
    assert frame[:4] == bytes([0x00, 0x02, 0x00, 0x03])


def test_checksum_is_xor():
    assert P.checksum(bytes([0x00, 0x01, 0x02, 0x03])) == 0x00
    assert P.checksum(bytes([0xFF, 0x0F])) == 0xF0


def test_flag_builders():
    assert P.serst() == 0xC0
    assert P.serst(ack=True) == 0xC8
    assert P.serst(more_pages=True, ack=True) == 0xCC
    assert P.tempo(2) == 0xE2
    assert P.function(P.Effect.APPEAR) == 0xC1
    assert P.page_status() == 0x80
    assert P.page_status(background=True) == 0xC0


def test_clock_page_layout():
    from datetime import datetime
    # Sunday 2026-10-11 13:05:09 -> dow 1
    p = P.clock_page(datetime(2026, 10, 11, 13, 5, 9))
    assert p.text == "130509111001"
    frame = P.build_frame(p, lines=4)
    assert frame[5:8] == b"000"
    assert frame[8:20] == b"130509111001"


def test_schedule_bit_must_match_block():
    import pytest
    with pytest.raises(ValueError):
        P.build_frame(P.Page(text="x"), lines=4, flags=P.serst(schedule=True))
