"""Пишет COFF-объект (.syso) с ресурсами Windows: иконка (RT_ICON + RT_GROUP_ICON) и манифест.
Go-линкер подхватывает *.syso из папки пакета сам."""
import struct, sys

def parse_ico(data):
    _, typ, count = struct.unpack_from("<HHH", data, 0)
    assert typ == 1
    out = []
    for i in range(count):
        w, h, cc, res, planes, bpp, size, off = struct.unpack_from("<BBBBHHII", data, 6 + 16 * i)
        out.append(dict(w=w, h=h, cc=cc, planes=planes, bpp=bpp, data=data[off:off + size]))
    return out

def build(ico_path, manifest_path, out_path):
    images = parse_ico(open(ico_path, "rb").read())
    manifest = open(manifest_path, "rb").read()
    group = struct.pack("<HHH", 0, 1, len(images))
    for i, im in enumerate(images, 1):
        group += struct.pack("<BBBBHHIH", im["w"], im["h"], im["cc"], 0, im["planes"] or 1, im["bpp"] or 32, len(im["data"]), i)
    # (тип, [(id, данные)])
    types = [(3, [(i, im["data"]) for i, im in enumerate(images, 1)]),
             (14, [(1, group)]),
             (24, [(1, manifest)])]
    LANG = 0x0409

    # раскладка: корень, каталоги имён, каталоги языков, записи данных, сами данные
    def dir_size(n): return 16 + 8 * n
    root_off = 0
    off = dir_size(len(types))
    name_offs = []
    for _, items in types:
        name_offs.append(off); off += dir_size(len(items))
    lang_offs = []
    for _, items in types:
        lo = []
        for _ in items:
            lo.append(off); off += dir_size(1)
        lang_offs.append(lo)
    entry_offs = []
    for _, items in types:
        eo = []
        for _ in items:
            eo.append(off); off += 16
        entry_offs.append(eo)
    data_offs = []
    for _, items in types:
        do = []
        for _, d in items:
            off = (off + 7) & ~7
            do.append(off); off += len(d)
        data_offs.append(do)
    total = (off + 7) & ~7

    buf = bytearray(total)
    def put_dir(at, entries):   # entries: [(id, offset, is_dir)]
        struct.pack_into("<IIHHHH", buf, at, 0, 0, 0, 0, 0, len(entries))
        for k, (eid, eoff, is_dir) in enumerate(entries):
            struct.pack_into("<II", buf, at + 16 + 8 * k, eid, eoff | (0x80000000 if is_dir else 0))
    put_dir(root_off, [(t, name_offs[ti], True) for ti, (t, _) in enumerate(types)])
    relocs = []
    for ti, (_, items) in enumerate(types):
        put_dir(name_offs[ti], [(rid, lang_offs[ti][k], True) for k, (rid, _) in enumerate(items)])
        for k, (rid, d) in enumerate(items):
            put_dir(lang_offs[ti][k], [(LANG, entry_offs[ti][k], False)])
            struct.pack_into("<IIII", buf, entry_offs[ti][k], data_offs[ti][k], len(d), 0, 0)
            relocs.append(entry_offs[ti][k])          # OffsetToData — RVA, нужна поправка
            buf[data_offs[ti][k]:data_offs[ti][k] + len(d)] = d

    n_sec = 1
    hdr = 20 + 40 * n_sec
    raw_ptr = hdr
    rel_ptr = raw_ptr + total
    sym_ptr = rel_ptr + 10 * len(relocs)
    out = bytearray()
    out += struct.pack("<HHIIIHH", 0x8664, n_sec, 0, sym_ptr, 1, 0, 0)
    out += struct.pack("<8sIIIIIIHHI", b".rsrc", 0, 0, total, raw_ptr, rel_ptr, 0, len(relocs), 0, 0x40000040)
    out += buf
    for r in relocs:
        out += struct.pack("<IIH", r, 0, 3)          # IMAGE_REL_AMD64_ADDR32NB к символу 0 (.rsrc)
    out += struct.pack("<8sIhHBB", b".rsrc", 0, 1, 0, 3, 0)
    out += struct.pack("<I", 4)                      # пустая таблица строк
    open(out_path, "wb").write(out)
    print(f"{out_path}: {len(out)} bytes, {len(images)} icons, manifest {len(manifest)} bytes")

if __name__ == "__main__":
    build(*sys.argv[1:4])
