avgdrop [![License](http://img.shields.io/:license-agpl3-blue.svg)](http://www.gnu.org/licenses/agpl-3.0.html)
=======

Bypass masks or dictionaries when the average crack rate drops below
a specified threshold. For use with [hashcat](https://hashcat.net/).
Linux-only.

## Install

    go install github.com/justpretending/avgdrop@latest

## Use

    avgdrop [OPTIONS] -- /path/to/hashcat [HASHCAT_OPTIONS]

Options:

`-min-avg N` (default 1.0) - the crack rate threshold (cracks per
minute).

`-d DELAY` (default 1m) - give that much time to hashcat at the
beginning and at the end of an attack. (The latter won't work
with piped input, obviously.)

### Examples

    avgdrop -- ./hashcat -m 0 -a 3 hashes masks.hcmask

    avgdrop -d 1m15s -- ./hashcat -m 0 hashes dictionaries/*.txt

## Donate

**Bitcoin (BTC):** `bc1q97urutztk4vjfe95vug6eg7wprqf5ec720kkj6`

**Litecoin (LTC):** `ltc1qdea4q6cyexp4zuccsrgdr6t50s6hn9azec3han`

**Monero (XMR):** `82wbi4A25w6MBZ8DJkpWuS3yQjzDdfy8H7PJNyFgdzdEGshCeZNhd5dETs4rmQD3GZN7gom8Cw1eL5ZjE61fGYHyVg2uMHn`
