struct String {
    char value[];
};

struct KSymHeader {
    u32 magic;
    u32 sym_count;
    u64 strings_size;    
};

struct KSymSymbol {
    u64 address;
    u8 type;
    u64 name_offset;
} [[format("symbol_format")]];

fn symbol_format(auto s) {
    String name @ 0x10 + header.sym_count * sizeof(KSymSymbol) + s.name_offset;
    return name.value;
};

KSymHeader header @ 0x0;
KSymSymbol symbols[header.sym_count] @ 0x10;