#!/usr/bin/env python3
"""Windows erőforrásfájl (.syso) generálása: ikon + verzióinfó + manifest.
Használat: make_res.py <verzió pl. 1.0.0> <kimeneti .syso> [ikon-előnézet.png]
"""
import io
import struct
import sys

from PIL import Image, ImageDraw

LANG = 0x0409

# ------------------------------------------------------------------ ikon


def draw_icon(size=1024):
    S = size
    # átlós színátmenet: kék -> türkiz
    grad = Image.new("RGB", (S, S))
    px = grad.load()
    c1, c2 = (31, 94, 220), (20, 184, 166)
    for y in range(S):
        for x in range(S):
            t = (x + y) / (2 * (S - 1))
            px[x, y] = tuple(int(c1[i] + (c2[i] - c1[i]) * t) for i in range(3))
    mask = Image.new("L", (S, S), 0)
    ImageDraw.Draw(mask).rounded_rectangle((0, 0, S - 1, S - 1), radius=int(S * 0.22), fill=255)
    img = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    img.paste(grad, (0, 0), mask)

    d = ImageDraw.Draw(img)
    W = (255, 255, 255, 255)
    k = S / 100.0

    # lista: három sor (pont + sáv)
    for i, w in enumerate((56, 46, 52)):
        y = 17 + i * 10.5
        d.ellipse((17 * k, y * k, 24 * k, (y + 7) * k), fill=W)
        d.rounded_rectangle((29 * k, (y + 0.8) * k, (29 + w) * k, (y + 6.2) * k), radius=2.6 * k, fill=W)

    # nyíl lefelé
    d.rectangle((46 * k, 50 * k, 54 * k, 60 * k), fill=W)
    d.polygon([(38 * k, 59 * k), (62 * k, 59 * k), (50 * k, 71 * k)], fill=W)

    # mappa
    d.rounded_rectangle((20 * k, 76 * k, 44 * k, 84 * k), radius=3 * k, fill=W)   # fül
    d.rounded_rectangle((20 * k, 79 * k, 80 * k, 92 * k), radius=3.5 * k, fill=W)  # test
    return img


def dib_icon_data(im):
    """32 bites BMP/DIB ikonkép (kis méretekhez)."""
    w, h = im.size
    im = im.convert("RGBA")
    px = im.load()
    pix = bytearray()
    for y in range(h - 1, -1, -1):  # alulról felfelé
        for x in range(w):
            r, g, b, a = px[x, y]
            pix += bytes((b, g, r, a))
    row = ((w + 31) // 32) * 4
    andmask = bytes(row * h)
    hdr = struct.pack("<IiiHHIIiiII", 40, w, h * 2, 1, 32, 0, len(pix) + len(andmask), 0, 0, 0, 0)
    return hdr + bytes(pix) + andmask


def png_data(im):
    b = io.BytesIO()
    im.save(b, "PNG", optimize=True)
    return b.getvalue()


# ------------------------------------------------------------------ verzióinfó


def pad4(b):
    return b + b"\x00" * ((-len(b)) % 4)


def u16z(s):
    return (s + "\x00").encode("utf-16-le")


def vblock(key, value=b"", wtype=1, children=(), vlen=None):
    head = u16z(key)
    body = pad4(struct.pack("<HHH", 0, 0, 0) + head)  # hossz később
    body = body[6:]  # csak a kulcs + pad (a fejléc külön)
    hdr_len = 6
    data = b"\x00" * hdr_len + body
    # a kulcs után 32 bites igazítás a fejléccel együtt
    data = pad4(b"\x00" * hdr_len + head)
    data += value
    if children:
        data = pad4(data)
        parts = []
        for i, c in enumerate(children):
            parts.append(c if i == len(children) - 1 else pad4(c))
        data += b"".join(parts)
    total = len(data)
    if vlen is None:
        vlen = 0
    return struct.pack("<HHH", total, vlen, wtype) + data[6:]


def version_info(ver):
    parts = [int(x) for x in ver.split(".")] + [0, 0, 0, 0]
    a, b, c, d = parts[:4]
    fv = f"{a}.{b}.{c}.{d}"
    ms, ls = (a << 16) | b, (c << 16) | d
    fixed = struct.pack("<13I", 0xFEEF04BD, 0x00010000, ms, ls, ms, ls, 0x3F, 0, 0x00040004, 1, 0, 0, 0)

    strings = [
        ("FileDescription", "Lejátszási lista másoló – m3u/m3u8 listák fájljainak másolása"),
        ("FileVersion", fv),
        ("InternalName", "PlaylistMasolo"),
        ("LegalCopyright", "Szabadon használható"),
        ("OriginalFilename", "PlaylistMasolo.exe"),
        ("ProductName", "Lejátszási lista másoló"),
        ("ProductVersion", f"{a}.{b}.{c}"),
    ]
    str_blocks = []
    for k, v in strings:
        val = u16z(v)
        str_blocks.append(vblock(k, val, 1, (), vlen=len(val) // 2))
    table = vblock("040E04B0", b"", 1, str_blocks)
    sfi = vblock("StringFileInfo", b"", 1, [table])
    var = vblock("Translation", struct.pack("<HH", 0x040E, 0x04B0), 0, (), vlen=4)
    vfi = vblock("VarFileInfo", b"", 1, [var])
    return vblock("VS_VERSION_INFO", fixed, 0, [sfi, vfi], vlen=len(fixed))


def manifest(ver):
    a, b, c = (ver.split(".") + ["0", "0"])[:3]
    return f"""<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
  <assemblyIdentity type="win32" name="PlaylistMasolo" version="{a}.{b}.{c}.0" processorArchitecture="*"/>
  <description>Lejátszási lista másoló</description>
  <dependency>
    <dependentAssembly>
      <assemblyIdentity type="win32" name="Microsoft.Windows.Common-Controls" version="6.0.0.0"
        processorArchitecture="*" publicKeyToken="6595b64144ccf1df" language="*"/>
    </dependentAssembly>
  </dependency>
  <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3">
    <security>
      <requestedPrivileges>
        <requestedExecutionLevel level="asInvoker" uiAccess="false"/>
      </requestedPrivileges>
    </security>
  </trustInfo>
  <compatibility xmlns="urn:schemas-microsoft-com:compatibility.v1">
    <application>
      <supportedOS Id="{{8e0f7a12-bfb3-4fe8-b9a5-48fd50a15a9a}}"/>
      <supportedOS Id="{{1f676c76-80e1-4239-95bb-83d0f6d0da78}}"/>
      <supportedOS Id="{{4a2f28e3-53b9-4441-ba9c-d69d4a4a6e38}}"/>
      <supportedOS Id="{{35138b9a-5d96-4fbd-8e2d-a2440225f93a}}"/>
      <supportedOS Id="{{e2011457-1546-43c5-a5fe-008deee3d3f0}}"/>
    </application>
  </compatibility>
  <application xmlns="urn:schemas-microsoft-com:asm.v3">
    <windowsSettings>
      <dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">true</dpiAware>
      <longPathAware xmlns="http://schemas.microsoft.com/SMI/2016/WindowsSettings">true</longPathAware>
    </windowsSettings>
  </application>
</assembly>
""".encode("utf-8")


# ------------------------------------------------------------------ erőforrás-fa + COFF


def build_rsrc(resources):
    """resources: {típus: {id: bytes}} -> (nyers .rsrc bájtok, [adatbejegyzés-mezők offsetjei, adat-offsetek])"""
    types = sorted(resources)
    # elrendezés
    off = 16 + 8 * len(types)
    type_dir_off = {}
    for t in types:
        type_dir_off[t] = off
        off += 16 + 8 * len(resources[t])
    lang_dir_off = {}
    for t in types:
        for i in sorted(resources[t]):
            lang_dir_off[(t, i)] = off
            off += 16 + 8
    entry_off = {}
    for t in types:
        for i in sorted(resources[t]):
            entry_off[(t, i)] = off
            off += 16
    data_off = {}
    for t in types:
        for i in sorted(resources[t]):
            off = (off + 7) & ~7
            data_off[(t, i)] = off
            off += len(resources[t][i])
    total = (off + 3) & ~3

    buf = bytearray(total)

    def dir_hdr(at, n):
        struct.pack_into("<IIHHHH", buf, at, 0, 0, 0, 0, 0, n)

    relocs = []
    dir_hdr(0, len(types))
    for k, t in enumerate(types):
        struct.pack_into("<II", buf, 16 + 8 * k, t, 0x80000000 | type_dir_off[t])
        dir_hdr(type_dir_off[t], len(resources[t]))
        for m, i in enumerate(sorted(resources[t])):
            struct.pack_into("<II", buf, type_dir_off[t] + 16 + 8 * m, i, 0x80000000 | lang_dir_off[(t, i)])
            dir_hdr(lang_dir_off[(t, i)], 1)
            struct.pack_into("<II", buf, lang_dir_off[(t, i)] + 16, LANG, entry_off[(t, i)])
            d = resources[t][i]
            struct.pack_into("<IIII", buf, entry_off[(t, i)], data_off[(t, i)], len(d), 0, 0)
            relocs.append(entry_off[(t, i)])
            buf[data_off[(t, i)]:data_off[(t, i)] + len(d)] = d
    return bytes(buf), relocs


def coff(rsrc, relocs, machine, rel_type):
    nrel = len(relocs)
    raw_ptr = 20 + 40
    rel_ptr = raw_ptr + len(rsrc)
    sym_ptr = rel_ptr + 10 * nrel
    out = struct.pack("<HHIIIHH", machine, 1, 0, sym_ptr, 2, 0, 0)
    out += struct.pack("<8sIIIIIIHHI", b".rsrc", 0, 0, len(rsrc), raw_ptr, rel_ptr, 0, nrel, 0, 0x40000040)
    out += rsrc
    for r in relocs:
        out += struct.pack("<IIH", r, 0, rel_type)
    out += struct.pack("<8sIhHBB", b".rsrc", 0, 1, 0, 3, 1)
    out += struct.pack("<IHHIHB3x", len(rsrc), nrel, 0, 0, 0, 0)
    out += struct.pack("<I", 4)
    return out


def main():
    ver = sys.argv[1]
    outp = sys.argv[2]
    prev = sys.argv[3] if len(sys.argv) > 3 else None

    big = draw_icon(1024)
    if prev:
        big.resize((256, 256), Image.LANCZOS).save(prev)

    sizes = [16, 24, 32, 48, 64, 256]
    icons = {}
    grp = struct.pack("<HHH", 0, 1, len(sizes))
    for n, s in enumerate(sizes, 1):
        im = big.resize((s, s), Image.LANCZOS)
        data = png_data(im) if s >= 256 else dib_icon_data(im)
        icons[n] = data
        grp += struct.pack("<BBBBHHIH", s % 256, s % 256, 0, 0, 1, 32, len(data), n)

    resources = {
        3: icons,                         # RT_ICON
        14: {1: grp},                     # RT_GROUP_ICON (azonosító: 1)
        16: {1: version_info(ver)},       # RT_VERSION
        24: {1: manifest(ver)},           # RT_MANIFEST
    }
    rsrc, relocs = build_rsrc(resources)
    # amd64: IMAGE_REL_AMD64_ADDR32NB = 3
    with open(outp, "wb") as f:
        f.write(coff(rsrc, relocs, 0x8664, 3))
    print("kész:", outp, len(rsrc), "bájt erőforrás,", len(relocs), "bejegyzés")


if __name__ == "__main__":
    main()
