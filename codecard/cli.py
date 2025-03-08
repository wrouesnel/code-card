# Utility to generate a random lookup code card.
# There is an option to generate a semi-random card which can map a known code to a given word.
import string
import random
import itertools
import collections
import hashlib
import secrets
from typing import Optional, IO

import click
from click.decorators import pass_meta_key

from reportlab.pdfbase import pdfmetrics
from reportlab.pdfgen.canvas import Canvas
from reportlab.lib.units import mm, cm, inch
from reportlab.pdfbase.pdfmetrics import stringWidth

from pprint import pprint

point = collections.namedtuple("point", ["x", "y"])

point.__add__ = lambda self, other: point(self.x + other.x, self.y + other.y)
point.__sub__ = lambda self, other: point(self.x - other.x, self.y - other.y)
point.__mul__ = lambda self, other: point(self.x * other, self.y * other)
point.__rmul__ = lambda self, other: point(self.x * other, self.y * other)


def get_string_height(font_family, font_size):
    face = pdfmetrics.getFont(font_family).face
    string_height = (face.ascent - face.descent) / 1000 * font_size
    return string_height


def render(code_table, output_name, title=None, debug_grid=False):
    FONT_FAMILY = "Courier"
    FONT_SIZE = 12

    PAGE_SIZE = point(56.0 * mm, 85.6 * mm)

    BORDER_MARGIN = point(1 * mm, 1 * mm)
    PADDING_LB = point(5 * mm, 13 * mm)
    PADDING_RT = point(5 * mm, 1 * mm)

    cut_border = (BORDER_MARGIN, PAGE_SIZE - 2 * BORDER_MARGIN)

    # Intentionally lop-sided since sleeves but the bottom off by more
    usable_rect = (BORDER_MARGIN + PADDING_LB, (PAGE_SIZE - PADDING_LB) - (2 * BORDER_MARGIN + PADDING_RT))

    canvas = Canvas(output_name, pagesize=PAGE_SIZE)
    canvas.setFont(FONT_FAMILY, FONT_SIZE)

    # Draw the cut border
    canvas.rect(cut_border[0].x, cut_border[0].y, cut_border[1].x, cut_border[1].y, stroke=1)

    if debug_grid:
        canvas.rect(usable_rect[0].x, usable_rect[0].y, usable_rect[1].x, usable_rect[1].y, stroke=1)

    # Get columns (fixed)
    COLUMNS = 2
    data_rows = list(itertools.batched(code_table.items(), n=COLUMNS))
    ROWS = len(data_rows) + 1  # Add one more row for the title

    column_width = usable_rect[1].x / COLUMNS
    row_height = usable_rect[1].y / ROWS

    font_height = get_string_height(FONT_FAMILY, FONT_SIZE)

    row_center_offset = row_height - (row_height - font_height - 2)  # This constant accounts for stroke border widths

    if title is not None:
        title_width = stringWidth(title, FONT_FAMILY, FONT_SIZE)

        canvas.drawString(usable_rect[0].x + (usable_rect[1].x / 2) - title_width / 2,
                          usable_rect[0].y + usable_rect[1].y - row_center_offset, title)

    canvas.line(usable_rect[0].x,
                usable_rect[0].y + usable_rect[1].y - row_height,
                usable_rect[0].x + usable_rect[1].x,
                usable_rect[0].y + usable_rect[1].y - row_height)

    for row_idx, values in enumerate(data_rows):
        row_idx += 1
        # canvas.rect(usable_rect[0].x+(column_width*col_idx),usable_rect[0].y - row_height + usable_rect[1].y - (row_height*row_idx), usable_rect[1].x, row_height)

        # values = itertools.chain(*values)
        for col_idx, value in enumerate(values):
            # Draw the columns
            if debug_grid:
                canvas.rect(usable_rect[0].x + (column_width * col_idx),
                            usable_rect[0].y - row_height + usable_rect[1].y - (row_height * row_idx), column_width,
                            row_height)

            text = f"{value[0]} {value[1] if isinstance(value[1], str) else ' '.join(value[1])}"
            text_width = stringWidth(text, FONT_FAMILY, FONT_SIZE)

            canvas.drawString(usable_rect[0].x + (usable_rect[1].x / 4) + (column_width * col_idx) - (text_width / 2),
                              (usable_rect[0].y + usable_rect[1].y) - (row_idx * row_height) - row_center_offset,
                              text)

    canvas.save()


# Setup the alphabet table - we loop through integer values of all digits, then randomly choose digits to fill the remainder
# to ensure diversity and balance repeats.
full_samples = len(string.ascii_uppercase) // len(string.digits)


def random_table():
    """Totally random table"""
    rand = random.SystemRandom()

    SRC_TABLE = {string.ascii_uppercase[idx]: string.digits[idx % 10] for idx in
                 range(full_samples * len(string.digits))}
    SRC_TABLE |= {letter: rand.choice(string.digits) for letter in
                  string.ascii_uppercase[full_samples * len(string.digits):]}

    return {k: v for k, v in zip(SRC_TABLE.keys(), rand.sample(list(SRC_TABLE.values()), len(SRC_TABLE)))}


def semirandom_table(salt, code, key):
    """Generate a semi-random table - fit a known numeric key to a codeword

    This works by matching digits to the letters. However, since digits can duplicate, we still achieve a decent level of entropy
    across multiple cards.
    """
    # Semi-random tables could leak information if multiple dis-similar cards generate for the same code/key combination.
    # Therefore although they are randomized, it is deterministic - the number generator is seeded with a hash of a salt, key and code.

    if len(salt.encode()) < 16:
        raise Exception(f"Please set a salt which is at least 16 bytes long (yours: {len(salt.encode())} bytes)")

    h = hashlib.sha256()
    h.update(salt.encode("utf8"))
    h.update(code.encode("utf8"))
    h.update(key.encode("utf8"))

    rand = random.Random(h.digest())

    SRC_TABLE = {string.ascii_uppercase[idx]: string.digits[idx % 10] for idx in
                 range(full_samples * len(string.digits))}
    SRC_TABLE |= {letter: rand.choice(string.digits) for letter in
                  string.ascii_uppercase[full_samples * len(string.digits):]}

    if len(key) < len(code):
        raise Exception("The password key must be at least as long as the code.")
    code_table = SRC_TABLE.copy()
    key = key.upper()

    counter = collections.Counter()
    for letter in key:
        counter[letter] += 1

    repeats = counter[max(counter, key=counter.get)]

    # Keep repeats sensibly limited (no more then 3)
    if repeats > 3:
        raise Exception(f"Too many repeated characters in the keyword: {repeats} > 3")

    # Build a table with random repeats in it up to max
    for letter in code_table:
        code_table[letter] = [code_table[letter]] + [rand.choice(string.digits) for _ in range(repeats - 1)]

    # Build the substitution for letters we *must* set. The rule is that letters are chosen based on the number of times
    # they're used - i.e. if O occurs twice, then the first usage is position 1, and the second usage is position 2.
    # We already know the maximum number of repeats from above, so all spaces are always populated.
    # IMPORTANT: this has a failure mode of bad codes - i.e. AAAAAA for 123456 would just output the whole pin somewhere
    # against the letter A, which drastically improves the odds (i.e. you have a 1 in 26 chance of just guessing correctly
    # since the whole pin is somewhere on the card).
    set_counter = collections.Counter()
    for digit, letter in zip(code, key):
        code_table[letter][set_counter[letter]] = digit
        set_counter[letter] += 1

    # for digit,letter in zip(code,key):

    return code_table

pass_output_file = pass_meta_key("output_file")
pass_title = pass_meta_key("card_title")

@click.group()
@click.option("--title", help="Title for the card")
@click.option("--output-file", type=click.File("wb"), default="code-card.pdf", help="Output filename", show_default=True)
@click.pass_context
def cli(ctx, title: Optional[str], output_file: IO[bytes]):
    ctx.meta["output_file"] = output_file
    ctx.meta["card_title"] = title


@cli.command("random")
@pass_output_file
@pass_title
def random_card(title, output_file):
    """Generate a new random code card"""
    code_table = random_table()
    render(code_table, output_file, title=title, debug_grid=False)


@cli.command("encoded")
@click.option("--salt", help="Salt value for code cards", show_default=True, default="1337code8Ackz0rs")
@click.password_option("--keyword", help="Keyword which will produce the code")
@click.password_option("--code", help="Numeric pin which will be encoded")
@pass_output_file
@pass_title
def encoded(title, output_file, salt: Optional[str], keyword: str, code: str):
    """Encode an existing PIN to a keyword on a code card"""
    if salt is None:
        salt = secrets.token_bytes(16)
    code_table = semirandom_table(salt, code, keyword)
    render(code_table, output_file, title=title, debug_grid=False)

def main():
    cli(auto_envvar_prefix="CODE_CARD")


if __name__ == "__main__":
    main()
