#!/usr/bin/env python3
import struct
import sys


def png_size(png_bytes: bytes):
    # PNG IHDR chunk starts at byte 16: width(4) height(4)
    if png_bytes[:8] != b"\x89PNG\r\n\x1a\n":
        raise ValueError("not a PNG")
    w = struct.unpack(">I", png_bytes[16:20])[0]
    h = struct.unpack(">I", png_bytes[20:24])[0]
    return w, h


def write_ico(out_path: str, png_paths: list[str]):
    images = []
    for p in png_paths:
        b = open(p, "rb").read()
        w, h = png_size(b)
        images.append((w, h, b))

    # sort by size ascending
    images.sort(key=lambda t: (t[0], t[1]))

    # ICONDIR
    out = bytearray()
    out += struct.pack("<HHH", 0, 1, len(images))

    # ICONDIRENTRY list
    entry_offset = 6 + 16 * len(images)
    image_offset = entry_offset
    entries = bytearray()
    blobs = bytearray()

    for (w, h, b) in images:
        ww = 0 if w >= 256 else w
        hh = 0 if h >= 256 else h
        bytes_in_res = len(b)
        entries += struct.pack(
            "<BBBBHHII",
            ww,
            hh,
            0,  # color count
            0,  # reserved
            1,  # planes
            32,  # bitcount
            bytes_in_res,
            image_offset,
        )
        blobs += b
        image_offset += bytes_in_res

    out += entries
    out += blobs
    with open(out_path, "wb") as f:
        f.write(out)


def main():
    if len(sys.argv) < 3:
        print("usage: make_ico.py OUT.ico in16.png in32.png ...", file=sys.stderr)
        sys.exit(2)
    out = sys.argv[1]
    ins = sys.argv[2:]
    write_ico(out, ins)


if __name__ == "__main__":
    main()


