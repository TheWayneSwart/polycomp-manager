import pytest
from polycomp.layout import layout, expand_markup, ROW_CHARS, PAIR_CHARS, TALL_VISIBLE


def test_four_normal_rows_are_padded_to_32():
    text, bold = layout(["ROW ONE", "ROW TWO", "ROW THREE", "ROW FOUR"])
    assert bold == 0
    assert text == "ROW ONE".ljust(32) + "ROW TWO".ljust(32) + "ROW THREE".ljust(32) + "ROW FOUR"


def test_tall_top_pair_consumes_64_chars():
    # verified live: 'HELLO WAYNE' padded to 64 put ROW THREE / ROW FOUR on rows 3 and 4
    text, bold = layout(["HELLO WAYNE", "ROW THREE", "ROW FOUR"], tall_top=True)
    assert bold == 0b01
    assert text == "HELLO WAYNE".ljust(PAIR_CHARS) + "ROW THREE".ljust(ROW_CHARS) + "ROW FOUR"


def test_both_pairs_tall_gives_two_rows():
    text, bold = layout(["TOP HALF", "BOTTOM HALF"], tall_top=True, tall_bottom=True)
    assert bold == 0b11
    assert text == "TOP HALF".ljust(PAIR_CHARS) + "BOTTOM HALF"


def test_tall_row_width_is_16():
    with pytest.raises(ValueError):
        layout(["X" * (TALL_VISIBLE + 1)], tall_top=True)
    layout(["X" * TALL_VISIBLE], tall_top=True)


def test_too_many_lines_rejected():
    with pytest.raises(ValueError):
        layout(["a", "b", "c", "d", "e"])
    with pytest.raises(ValueError):
        layout(["a", "b", "c", "d"], tall_top=True)


def test_markup_takes_no_room():
    text, visible = expand_markup("{G}GREEN{D} x")
    assert text == "\x1cGGREEN\x1cD x" and visible == 7
    out, _ = layout(["{Y}" + "Y" * ROW_CHARS, "next"])
    assert out.startswith("\x1cY" + "Y" * ROW_CHARS + "\x1cDnext")  # reset code, zero padding


def test_blank_line_keeps_row_empty():
    text, _ = layout(["ROW ONE", "", "ROW THREE"])
    assert text == "ROW ONE".ljust(32) + " " * 32 + "ROW THREE"


def test_centre_pads_with_leading_spaces_and_keeps_row_width():
    text, _ = layout(["Bite and Bean", "{G}cappuccino - R35.00"], tall_top=True, centre=True)
    tall, rest = text[:64], text[64:]
    assert tall == " " + "Bite and Bean" + " " * 50          # (16-13)//2 = 1 leading, 64 total
    assert rest == " " * 6 + "\x1cG" + "cappuccino - R35.00"  # (32-19)//2 = 6 leading, last line unpadded


def test_enlarge_counts_double_width_cells():
    _, cells = expand_markup("{E}Cappuccino")
    assert cells == 20
    _, cells = expand_markup("{E}ab{D}cd")
    assert cells == 6
    text, _ = layout(["{E}Cappuccino", "x"], centre=True)
    assert text.startswith(" " * 6 + "\x1cECappuccino")   # (32-20)//2 = 6
    with pytest.raises(ValueError):
        layout(["{E}" + "x" * 17])                           # 34 cells > 32


def test_right_align_pads_to_row_edge():
    text, _ = layout(["Cappuccino R35", "{G}Bean & Bite"], tall_top=True, tall_bottom=True, align="right")
    assert text[:64] == " " * 2 + "Cappuccino R35" + " " * 48   # 16-14 = 2 leading
    assert text[64:] == " " * 5 + "\x1cGBean & Bite"             # 16-11 = 5 leading, last line unpadded
    with pytest.raises(ValueError):
        layout(["x"], align="middle")


def test_formatting_is_reset_before_padding():
    text, _ = layout(["{E}Cappuccino", "R35.00"])
    assert text == "\x1cECappuccino\x1cD" + " " * 12 + "R35.00"   # 20 wide cells + 12 pad = 32
    text, _ = layout(["plain", "next"])
    assert text == "plain" + " " * 27 + "next"
